// Package tray provides a macOS system tray interface for the Kuchipudi gesture recognition system.
package tray

import (
	"sync"

	"github.com/getlantern/systray"
)

// Tray represents the macOS system tray application.
type Tray struct {
	onToggle   func(enabled bool)
	onSettings func()
	onQuit     func()
	enabled    bool
	mu         sync.RWMutex

	// Menu items stored for later updates
	menuToggle      *systray.MenuItem
	menuLastGesture *systray.MenuItem
}

// New creates a new Tray instance with enabled state set to true by default.
func New() *Tray {
	return &Tray{
		enabled: true,
	}
}

// OnToggle sets the callback function to be called when the enabled state is toggled.
func (t *Tray) OnToggle(fn func(enabled bool)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onToggle = fn
}

// OnSettings sets the callback function to be called when the settings menu item is clicked.
func (t *Tray) OnSettings(fn func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onSettings = fn
}

// OnQuit sets the callback function to be called when the quit menu item is clicked.
func (t *Tray) OnQuit(fn func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onQuit = fn
}

// Run starts the system tray application.
// This function blocks until systray.Quit() is called.
func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

// onReady is called when the system tray is ready.
// It sets up the menu structure.
func (t *Tray) onReady() {
	// Set the tray title and tooltip
	systray.SetTitle("Kuchipudi")
	systray.SetTooltip("Kuchipudi Gesture Recognition")

	// Create menu items
	t.menuToggle = systray.AddMenuItem("● Enabled", "Toggle gesture recognition")
	systray.AddSeparator()

	t.menuLastGesture = systray.AddMenuItem("Last: none", "Last detected gesture")
	t.menuLastGesture.Disable()
	systray.AddSeparator()

	menuSettings := systray.AddMenuItem("Open Settings...", "Open settings in browser")
	systray.AddSeparator()

	menuQuit := systray.AddMenuItem("Quit", "Quit Kuchipudi")

	// Handle menu item clicks in a separate goroutine
	go func() {
		for {
			select {
			case <-t.menuToggle.ClickedCh:
				t.handleToggle()
			case <-menuSettings.ClickedCh:
				t.handleSettings()
			case <-menuQuit.ClickedCh:
				t.handleQuit()
				return
			}
		}
	}()
}

// onExit is called when the system tray is about to exit.
// It performs cleanup tasks.
func (t *Tray) onExit() {
	// Cleanup resources if needed
}

// handleToggle handles the toggle menu item click.
func (t *Tray) handleToggle() {
	t.mu.Lock()
	t.enabled = !t.enabled
	enabled := t.enabled

	// Update menu item text based on new state
	if enabled {
		t.menuToggle.SetTitle("● Enabled")
	} else {
		t.menuToggle.SetTitle("○ Disabled")
	}

	callback := t.onToggle
	t.mu.Unlock()

	// Call the callback outside the lock to prevent deadlocks
	if callback != nil {
		callback(enabled)
	}
}

// handleSettings handles the settings menu item click.
func (t *Tray) handleSettings() {
	t.mu.RLock()
	callback := t.onSettings
	t.mu.RUnlock()

	if callback != nil {
		callback()
	}
}

// handleQuit handles the quit menu item click.
func (t *Tray) handleQuit() {
	t.mu.RLock()
	callback := t.onQuit
	t.mu.RUnlock()

	if callback != nil {
		callback()
	}

	systray.Quit()
}

// SetLastGesture updates the last gesture display in the menu.
func (t *Tray) SetLastGesture(name string) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.menuLastGesture != nil {
		if name == "" {
			t.menuLastGesture.SetTitle("Last: none")
		} else {
			t.menuLastGesture.SetTitle("Last: " + name)
		}
	}
}

// IsEnabled returns the current enabled state.
func (t *Tray) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}
