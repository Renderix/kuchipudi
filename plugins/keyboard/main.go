// Package main provides a keyboard plugin for macOS.
// It sends keyboard shortcuts and keystrokes via AppleScript.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Request represents the input from the plugin executor.
type Request struct {
	Action  string          `json:"action"`
	Gesture string          `json:"gesture"`
	Config  json.RawMessage `json:"config"`
	Params  json.RawMessage `json:"params"`
}

// Response represents the output to the plugin executor.
type Response struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// KeystrokeParams defines parameters for keystroke and shortcut actions.
type KeystrokeParams struct {
	Key       string   `json:"key"`
	Modifiers []string `json:"modifiers"` // command, option, control, shift
}

// modifierMap maps user-friendly modifier names to AppleScript equivalents.
var modifierMap = map[string]string{
	"command": "command down",
	"cmd":     "command down",
	"option":  "option down",
	"alt":     "option down",
	"control": "control down",
	"ctrl":    "control down",
	"shift":   "shift down",
}

func main() {
	// Read request from stdin
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeErrorResponse(fmt.Sprintf("failed to decode request: %v", err))
		return
	}

	// Handle keystroke and shortcut actions
	switch req.Action {
	case "keystroke", "shortcut":
		if err := handleKeystroke(req.Params); err != nil {
			writeErrorResponse(fmt.Sprintf("action %s failed: %v", req.Action, err))
			return
		}
	default:
		writeErrorResponse(fmt.Sprintf("unknown action: %s", req.Action))
		return
	}

	// Write success response
	writeSuccessResponse()
}

// handleKeystroke processes keystroke and shortcut actions.
func handleKeystroke(params json.RawMessage) error {
	var p KeystrokeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Errorf("failed to parse params: %w", err)
	}

	if p.Key == "" {
		return fmt.Errorf("key is required")
	}

	script := buildKeystrokeScript(p.Key, p.Modifiers)
	return runAppleScript(script)
}

// buildKeystrokeScript generates an AppleScript for the given key and modifiers.
func buildKeystrokeScript(key string, modifiers []string) string {
	if len(modifiers) == 0 {
		return fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, key)
	}

	// Convert modifiers to AppleScript format
	var appleModifiers []string
	for _, mod := range modifiers {
		if appleMod, ok := modifierMap[strings.ToLower(mod)]; ok {
			appleModifiers = append(appleModifiers, appleMod)
		}
	}

	if len(appleModifiers) == 0 {
		return fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, key)
	}

	modifierList := strings.Join(appleModifiers, ", ")
	return fmt.Sprintf(`tell application "System Events" to keystroke "%s" using {%s}`, key, modifierList)
}

// writeErrorResponse writes an error response to stdout.
func writeErrorResponse(errMsg string) {
	resp := Response{
		Success: false,
		Error:   errMsg,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

// writeSuccessResponse writes a success response to stdout.
func writeSuccessResponse() {
	resp := Response{
		Success: true,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

// runAppleScript executes an AppleScript command and returns any error.
func runAppleScript(script string) error {
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}
