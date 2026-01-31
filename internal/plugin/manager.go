package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// ErrPluginNotFound is returned when a requested plugin cannot be found.
var ErrPluginNotFound = errors.New("plugin not found")

// Manager manages plugin discovery and access.
type Manager struct {
	pluginDir string
	plugins   map[string]*Plugin
	mu        sync.RWMutex
}

// NewManager creates a new plugin Manager with the given plugin directory.
func NewManager(pluginDir string) *Manager {
	return &Manager{
		pluginDir: pluginDir,
		plugins:   make(map[string]*Plugin),
	}
}

// Discover scans the plugin directory for plugin.json files and loads them.
// Each subdirectory in the plugin directory is expected to be a plugin with a plugin.json manifest.
func (m *Manager) Discover() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing plugins
	m.plugins = make(map[string]*Plugin)

	// Check if plugin directory exists
	info, err := os.Stat(m.pluginDir)
	if os.IsNotExist(err) {
		return nil // No plugins directory, nothing to discover
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil // Not a directory, nothing to discover
	}

	// Read plugin directory entries
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(m.pluginDir, entry.Name())
		manifestPath := filepath.Join(pluginPath, "plugin.json")

		// Check if plugin.json exists
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		// Read and parse the manifest
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // Skip plugins we can't read
		}

		var manifest Manifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			continue // Skip plugins with invalid JSON
		}

		// Determine the executable path
		executablePath := filepath.Join(pluginPath, manifest.Executable)

		plugin := &Plugin{
			Manifest:   manifest,
			Path:       pluginPath,
			Executable: executablePath,
		}

		m.plugins[manifest.Name] = plugin
	}

	return nil
}

// Get returns a plugin by name.
// Returns ErrPluginNotFound if the plugin does not exist.
func (m *Manager) Get(name string) (*Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, ok := m.plugins[name]
	if !ok {
		return nil, ErrPluginNotFound
	}

	return plugin, nil
}

// List returns a slice of all discovered plugins.
func (m *Manager) List() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// PluginDir returns the plugin directory path.
func (m *Manager) PluginDir() string {
	return m.pluginDir
}
