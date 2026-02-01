package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPlugin_SystemControl_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if runtime.GOOS != "darwin" {
		t.Skip("system-control plugin only works on macOS")
	}

	// Find the built plugin
	pluginDir := findPluginDir("system-control")
	if pluginDir == "" {
		t.Skip("system-control plugin not built")
	}

	mgr := NewManager(filepath.Dir(pluginDir))
	if err := mgr.Discover(); err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	plug, err := mgr.Get("system-control")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	executor := NewExecutor(5000)

	// Test with an action that doesn't have side effects
	// (volume-mute toggles, so we'd need to restore state)
	// Instead, test with an invalid action to verify error handling
	req := &Request{
		Action: "execute",
		Params: json.RawMessage(`{"action_name": "invalid-action"}`),
	}

	resp, err := executor.Execute(plug, req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if resp.Success {
		t.Error("expected failure for invalid action")
	}
}

func TestPlugin_Keyboard_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if runtime.GOOS != "darwin" {
		t.Skip("keyboard plugin only works on macOS")
	}

	pluginDir := findPluginDir("keyboard")
	if pluginDir == "" {
		t.Skip("keyboard plugin not built")
	}

	mgr := NewManager(filepath.Dir(pluginDir))
	mgr.Discover()

	plug, err := mgr.Get("keyboard")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	executor := NewExecutor(5000)

	// Test with missing key parameter
	req := &Request{
		Action: "execute",
		Params: json.RawMessage(`{"key": ""}`),
	}

	resp, err := executor.Execute(plug, req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if resp.Success {
		t.Error("expected failure for empty key")
	}
}

func findPluginDir(name string) string {
	candidates := []string{
		filepath.Join("../../plugins", name),
		filepath.Join("../../../plugins", name),
	}

	for _, dir := range candidates {
		manifest := filepath.Join(dir, "plugin.json")
		if _, err := os.Stat(manifest); err == nil {
			return dir
		}
	}
	return ""
}
