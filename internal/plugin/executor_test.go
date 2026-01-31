package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecutor_Execute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Create a temporary directory for the test plugin
	tmpDir, err := os.MkdirTemp("", "kuchipudi-executor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shell script that echoes a success JSON response
	scriptContent := `#!/bin/sh
cat <<'EOF'
{"success":true,"data":{"message":"hello world"}}
EOF
`
	scriptPath := filepath.Join(tmpDir, "test-plugin.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create a plugin
	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "test-plugin",
			Version:    "1.0.0",
			Executable: "test-plugin.sh",
			Actions:    []string{"test-action"},
		},
		Path:       tmpDir,
		Executable: scriptPath,
	}

	// Create a request
	request := &Request{
		Action:  "test-action",
		Gesture: "swipe-left",
		Config:  json.RawMessage(`{"key":"value"}`),
		Params:  json.RawMessage(`{"param1":"value1"}`),
	}

	// Create executor and execute
	executor := NewExecutor(5000) // 5 second timeout
	response, err := executor.Execute(plugin, request)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify response
	if !response.Success {
		t.Errorf("expected success=true, got false")
	}
	if response.Error != "" {
		t.Errorf("expected empty error, got %q", response.Error)
	}

	// Verify data
	var data map[string]interface{}
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal response data: %v", err)
	}
	if data["message"] != "hello world" {
		t.Errorf("expected message 'hello world', got %v", data["message"])
	}
}

func TestExecutor_Execute_ReadsStdin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Create a temporary directory for the test plugin
	tmpDir, err := os.MkdirTemp("", "kuchipudi-executor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shell script that reads stdin and echoes it back in the response
	scriptContent := `#!/bin/sh
INPUT=$(cat)
echo "{\"success\":true,\"data\":{\"received\":$INPUT}}"
`
	scriptPath := filepath.Join(tmpDir, "echo-plugin.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create a plugin
	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "echo-plugin",
			Version:    "1.0.0",
			Executable: "echo-plugin.sh",
			Actions:    []string{"echo"},
		},
		Path:       tmpDir,
		Executable: scriptPath,
	}

	// Create a request with specific values
	request := &Request{
		Action:  "echo",
		Gesture: "tap",
		Config:  json.RawMessage(`{"setting":"enabled"}`),
		Params:  json.RawMessage(`{"count":42}`),
	}

	// Create executor and execute
	executor := NewExecutor(5000)
	response, err := executor.Execute(plugin, request)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if !response.Success {
		t.Errorf("expected success=true, got false")
	}

	// Verify the request was received
	var data map[string]interface{}
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal response data: %v", err)
	}

	received, ok := data["received"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'received' to be an object, got %T", data["received"])
	}

	if received["action"] != "echo" {
		t.Errorf("expected action 'echo', got %v", received["action"])
	}
	if received["gesture"] != "tap" {
		t.Errorf("expected gesture 'tap', got %v", received["gesture"])
	}
}

func TestExecutor_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Create a temporary directory for the test plugin
	tmpDir, err := os.MkdirTemp("", "kuchipudi-executor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shell script that sleeps longer than the timeout
	scriptContent := `#!/bin/sh
sleep 10
echo '{"success":true}'
`
	scriptPath := filepath.Join(tmpDir, "slow-plugin.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create a plugin
	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "slow-plugin",
			Version:    "1.0.0",
			Executable: "slow-plugin.sh",
			Actions:    []string{"slow"},
		},
		Path:       tmpDir,
		Executable: scriptPath,
	}

	// Create a request
	request := &Request{
		Action:  "slow",
		Gesture: "swipe",
	}

	// Create executor with a very short timeout (100ms)
	executor := NewExecutor(100)
	_, err = executor.Execute(plugin, request)

	// Should return a timeout error
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "killed") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected timeout-related error, got: %v", err)
	}
}

func TestExecutor_Execute_ErrorResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Create a temporary directory for the test plugin
	tmpDir, err := os.MkdirTemp("", "kuchipudi-executor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shell script that returns an error response
	scriptContent := `#!/bin/sh
echo '{"success":false,"error":"something went wrong"}'
`
	scriptPath := filepath.Join(tmpDir, "error-plugin.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create a plugin
	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "error-plugin",
			Version:    "1.0.0",
			Executable: "error-plugin.sh",
			Actions:    []string{"fail"},
		},
		Path:       tmpDir,
		Executable: scriptPath,
	}

	// Create a request
	request := &Request{
		Action:  "fail",
		Gesture: "tap",
	}

	// Create executor and execute
	executor := NewExecutor(5000)
	response, err := executor.Execute(plugin, request)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify error response
	if response.Success {
		t.Errorf("expected success=false, got true")
	}
	if response.Error != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got %q", response.Error)
	}
}

func TestExecutor_Execute_InvalidJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Create a temporary directory for the test plugin
	tmpDir, err := os.MkdirTemp("", "kuchipudi-executor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shell script that outputs invalid JSON
	scriptContent := `#!/bin/sh
echo 'not valid json'
`
	scriptPath := filepath.Join(tmpDir, "bad-plugin.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create a plugin
	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "bad-plugin",
			Version:    "1.0.0",
			Executable: "bad-plugin.sh",
			Actions:    []string{"bad"},
		},
		Path:       tmpDir,
		Executable: scriptPath,
	}

	// Create a request
	request := &Request{
		Action:  "bad",
		Gesture: "tap",
	}

	// Create executor and execute
	executor := NewExecutor(5000)
	_, err = executor.Execute(plugin, request)

	// Should return an error for invalid JSON
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestExecutor_Execute_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Create a temporary directory for the test plugin
	tmpDir, err := os.MkdirTemp("", "kuchipudi-executor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shell script that exits with non-zero status
	scriptContent := `#!/bin/sh
echo "Error: something failed" >&2
exit 1
`
	scriptPath := filepath.Join(tmpDir, "exit-plugin.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Create a plugin
	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "exit-plugin",
			Version:    "1.0.0",
			Executable: "exit-plugin.sh",
			Actions:    []string{"exit"},
		},
		Path:       tmpDir,
		Executable: scriptPath,
	}

	// Create a request
	request := &Request{
		Action:  "exit",
		Gesture: "tap",
	}

	// Create executor and execute
	executor := NewExecutor(5000)
	_, err = executor.Execute(plugin, request)

	// Should return an error for non-zero exit
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
}

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor(3000)
	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}
	if executor.timeoutMs != 3000 {
		t.Errorf("expected timeoutMs=3000, got %d", executor.timeoutMs)
	}
}
