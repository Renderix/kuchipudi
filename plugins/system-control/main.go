// Package main provides a system control plugin for macOS.
// It handles volume, brightness, and media playback controls via AppleScript.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// actionHandler defines a function type for handling specific actions.
type actionHandler func() error

// actionHandlers maps action names to their handler functions.
var actionHandlers = map[string]actionHandler{
	"volume-up":        volumeUp,
	"volume-down":      volumeDown,
	"volume-mute":      volumeMute,
	"brightness-up":    brightnessUp,
	"brightness-down":  brightnessDown,
	"media-play-pause": mediaPlayPause,
	"media-next":       mediaNext,
	"media-prev":       mediaPrev,
}

func main() {
	// Read request from stdin
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeErrorResponse(fmt.Sprintf("failed to decode request: %v", err))
		return
	}

	// Look up the handler for the action
	handler, ok := actionHandlers[req.Action]
	if !ok {
		writeErrorResponse(fmt.Sprintf("unknown action: %s", req.Action))
		return
	}

	// Execute the handler
	if err := handler(); err != nil {
		writeErrorResponse(fmt.Sprintf("action %s failed: %v", req.Action, err))
		return
	}

	// Write success response
	writeSuccessResponse()
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

// volumeUp increases the system volume by 10%.
func volumeUp() error {
	script := `set volume output volume ((output volume of (get volume settings)) + 10)`
	return runAppleScript(script)
}

// volumeDown decreases the system volume by 10%.
func volumeDown() error {
	script := `set volume output volume ((output volume of (get volume settings)) - 10)`
	return runAppleScript(script)
}

// volumeMute toggles the system mute state.
func volumeMute() error {
	script := `set volume output muted (not (output muted of (get volume settings)))`
	return runAppleScript(script)
}

// brightnessUp increases the screen brightness.
func brightnessUp() error {
	script := `tell application "System Events"
	key code 144
end tell`
	return runAppleScript(script)
}

// brightnessDown decreases the screen brightness.
func brightnessDown() error {
	script := `tell application "System Events"
	key code 145
end tell`
	return runAppleScript(script)
}

// mediaPlayPause toggles media play/pause using the F8/Play-Pause media key.
func mediaPlayPause() error {
	script := `tell application "System Events"
	key code 100
end tell`
	return runAppleScript(script)
}

// mediaNext skips to the next track using the F9/Next media key.
func mediaNext() error {
	script := `tell application "System Events"
	key code 101
end tell`
	return runAppleScript(script)
}

// mediaPrev skips to the previous track using the F7/Previous media key.
func mediaPrev() error {
	script := `tell application "System Events"
	key code 98
end tell`
	return runAppleScript(script)
}
