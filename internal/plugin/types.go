// Package plugin provides plugin management and execution capabilities for the Kuchipudi gesture recognition system.
package plugin

import "encoding/json"

// Manifest describes a plugin's metadata and capabilities.
type Manifest struct {
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	Executable   string          `json:"executable"`
	Actions      []string        `json:"actions"`
	ConfigSchema json.RawMessage `json:"configSchema,omitempty"`
}

// Request represents a request sent to a plugin for execution.
type Request struct {
	Action  string          `json:"action"`
	Gesture string          `json:"gesture"`
	Config  json.RawMessage `json:"config"`
	Params  json.RawMessage `json:"params"`
}

// Response represents the response from a plugin execution.
type Response struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Plugin represents a discovered plugin with its manifest and location.
type Plugin struct {
	Manifest   Manifest
	Path       string
	Executable string
}
