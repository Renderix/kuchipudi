package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestManager_Discover(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir, err := os.MkdirTemp("", "kuchipudi-plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test plugin directory
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	// Create a plugin.json manifest
	manifest := Manifest{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Executable:  "test-plugin",
		Actions:     []string{"action1", "action2"},
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.json")
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create the manager and discover plugins
	manager := NewManager(tmpDir)
	if err := manager.Discover(); err != nil {
		t.Fatalf("Discover() failed: %v", err)
	}

	// Verify the plugin was discovered
	plugins := manager.List()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	// Verify plugin details
	plugin := plugins[0]
	if plugin.Manifest.Name != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got %q", plugin.Manifest.Name)
	}
	if plugin.Manifest.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", plugin.Manifest.Version)
	}
	if plugin.Manifest.Description != "A test plugin" {
		t.Errorf("expected description 'A test plugin', got %q", plugin.Manifest.Description)
	}
	if len(plugin.Manifest.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(plugin.Manifest.Actions))
	}
	if plugin.Path != pluginDir {
		t.Errorf("expected path %q, got %q", pluginDir, plugin.Path)
	}
}

func TestManager_Discover_MultiplePlugins(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir, err := os.MkdirTemp("", "kuchipudi-plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create two test plugins
	for _, name := range []string{"plugin-a", "plugin-b"} {
		pluginDir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatalf("failed to create plugin dir: %v", err)
		}

		manifest := Manifest{
			Name:       name,
			Version:    "1.0.0",
			Executable: name,
			Actions:    []string{"action"},
		}

		manifestBytes, err := json.Marshal(manifest)
		if err != nil {
			t.Fatalf("failed to marshal manifest: %v", err)
		}

		manifestPath := filepath.Join(pluginDir, "plugin.json")
		if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}
	}

	// Create the manager and discover plugins
	manager := NewManager(tmpDir)
	if err := manager.Discover(); err != nil {
		t.Fatalf("Discover() failed: %v", err)
	}

	// Verify both plugins were discovered
	plugins := manager.List()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
}

func TestManager_Discover_EmptyDir(t *testing.T) {
	// Create a temporary empty plugin directory
	tmpDir, err := os.MkdirTemp("", "kuchipudi-plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the manager and discover plugins
	manager := NewManager(tmpDir)
	if err := manager.Discover(); err != nil {
		t.Fatalf("Discover() failed on empty dir: %v", err)
	}

	// Should have no plugins
	plugins := manager.List()
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestManager_Get(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir, err := os.MkdirTemp("", "kuchipudi-plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test plugin
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	manifest := Manifest{
		Name:       "my-plugin",
		Version:    "2.0.0",
		Executable: "my-plugin-bin",
		Actions:    []string{"run"},
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.json")
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create the manager and discover plugins
	manager := NewManager(tmpDir)
	if err := manager.Discover(); err != nil {
		t.Fatalf("Discover() failed: %v", err)
	}

	// Get the plugin by name
	plugin, err := manager.Get("my-plugin")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if plugin.Manifest.Name != "my-plugin" {
		t.Errorf("expected plugin name 'my-plugin', got %q", plugin.Manifest.Name)
	}
	if plugin.Manifest.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", plugin.Manifest.Version)
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir, err := os.MkdirTemp("", "kuchipudi-plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the manager (empty)
	manager := NewManager(tmpDir)

	// Try to get a non-existent plugin
	_, err = manager.Get("nonexistent-plugin")
	if err != ErrPluginNotFound {
		t.Errorf("expected ErrPluginNotFound, got %v", err)
	}
}

func TestManager_PluginDir(t *testing.T) {
	pluginDir := "/path/to/plugins"
	manager := NewManager(pluginDir)

	if manager.PluginDir() != pluginDir {
		t.Errorf("expected plugin dir %q, got %q", pluginDir, manager.PluginDir())
	}
}

func TestManager_Discover_InvalidJSON(t *testing.T) {
	// Create a temporary plugin directory
	tmpDir, err := os.MkdirTemp("", "kuchipudi-plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a plugin directory with invalid JSON
	pluginDir := filepath.Join(tmpDir, "bad-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.json")
	if err := os.WriteFile(manifestPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create the manager and discover plugins
	manager := NewManager(tmpDir)

	// Discover should skip invalid plugins gracefully
	if err := manager.Discover(); err != nil {
		t.Fatalf("Discover() failed unexpectedly: %v", err)
	}

	// Should have no valid plugins
	plugins := manager.List()
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins (invalid JSON should be skipped), got %d", len(plugins))
	}
}

func TestManager_Discover_NonExistentDir(t *testing.T) {
	// Create a manager with non-existent directory
	manager := NewManager("/path/that/does/not/exist")

	// Discover should not fail, just return empty list
	if err := manager.Discover(); err != nil {
		t.Fatalf("Discover() failed on non-existent dir: %v", err)
	}

	plugins := manager.List()
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}
