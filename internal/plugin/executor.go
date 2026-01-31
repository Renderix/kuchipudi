package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// Executor handles the execution of plugins with timeout support.
type Executor struct {
	timeoutMs int
}

// NewExecutor creates a new Executor with the specified timeout in milliseconds.
func NewExecutor(timeoutMs int) *Executor {
	return &Executor{
		timeoutMs: timeoutMs,
	}
}

// Execute runs a plugin with the given request and returns the response.
// It creates a context with the configured timeout, marshals the request to JSON,
// sends it to the plugin via stdin, and parses the stdout as a Response.
func (e *Executor) Execute(plugin *Plugin, req *Request) (*Response, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.timeoutMs)*time.Millisecond)
	defer cancel()

	// Create command with context
	cmd := exec.CommandContext(ctx, plugin.Executable)

	// Set working directory to plugin path
	cmd.Dir = plugin.Path

	// Marshal request to JSON
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Set up stdin with the request JSON
	cmd.Stdin = bytes.NewReader(reqJSON)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Check for context deadline exceeded (timeout)
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("plugin execution timeout after %dms", e.timeoutMs)
	}

	// Check for execution error
	if err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("plugin execution failed: %w, stderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("plugin execution failed: %w", err)
	}

	// Parse the response from stdout
	var response Response
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to parse plugin response: %w, stdout: %s", err, stdout.String())
	}

	return &response, nil
}
