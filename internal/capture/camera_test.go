package capture

import (
	"testing"
)

func TestNewCamera(t *testing.T) {
	tests := []struct {
		name     string
		deviceID int
		wantFPS  int
	}{
		{
			name:     "default device",
			deviceID: 0,
			wantFPS:  5,
		},
		{
			name:     "device 1",
			deviceID: 1,
			wantFPS:  5,
		},
		{
			name:     "device 2",
			deviceID: 2,
			wantFPS:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cam := NewCamera(tt.deviceID)

			if cam == nil {
				t.Fatal("NewCamera returned nil")
			}

			// Check default FPS through public interface
			if got := cam.FPS(); got != tt.wantFPS {
				t.Errorf("FPS() = %d, want %d (default)", got, tt.wantFPS)
			}

			// Camera should not be running initially
			if cam.IsOpen() {
				t.Error("camera should not be running initially")
			}
		})
	}
}

func TestCamera_SetFPS(t *testing.T) {
	cam := NewCamera(0)

	tests := []struct {
		name    string
		fps     int
		wantFPS int
	}{
		{
			name:    "set to 10",
			fps:     10,
			wantFPS: 10,
		},
		{
			name:    "set to 30",
			fps:     30,
			wantFPS: 30,
		},
		{
			name:    "set to 1",
			fps:     1,
			wantFPS: 1,
		},
		{
			name:    "set to 0 should keep previous",
			fps:     0,
			wantFPS: 1, // Previous value
		},
		{
			name:    "set to negative should keep previous",
			fps:     -5,
			wantFPS: 1, // Previous value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cam.SetFPS(tt.fps)

			got := cam.FPS()
			if got != tt.wantFPS {
				t.Errorf("FPS() = %d, want %d", got, tt.wantFPS)
			}
		})
	}
}

func TestCamera_FPS(t *testing.T) {
	cam := NewCamera(0)

	// Default FPS
	if got := cam.FPS(); got != 5 {
		t.Errorf("FPS() = %d, want 5 (default)", got)
	}

	// After setting
	cam.SetFPS(15)
	if got := cam.FPS(); got != 15 {
		t.Errorf("FPS() = %d, want 15", got)
	}
}

func TestCamera_IsOpen_NotOpened(t *testing.T) {
	cam := NewCamera(0)

	if cam.IsOpen() {
		t.Error("IsOpen() should return false before Open() is called")
	}
}

func TestCamera_OpenClose_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cam := NewCamera(0)

	// Test Open
	err := cam.Open()
	if err != nil {
		t.Skipf("skipping test - camera not available: %v", err)
	}

	if !cam.IsOpen() {
		t.Error("IsOpen() should return true after Open()")
	}

	// Test ReadFrame
	mat, err := cam.ReadFrame()
	if err != nil {
		t.Errorf("ReadFrame() failed: %v", err)
	} else {
		if mat == nil {
			t.Error("ReadFrame() returned nil mat")
		} else if mat.Empty() {
			t.Error("ReadFrame() returned empty mat")
		} else {
			// Verify dimensions (we set 640x480)
			if mat.Cols() != 640 || mat.Rows() != 480 {
				t.Logf("Frame dimensions: %dx%d (expected 640x480, but camera may not support)", mat.Cols(), mat.Rows())
			}
			mat.Close()
		}
	}

	// Test Close
	err = cam.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if cam.IsOpen() {
		t.Error("IsOpen() should return false after Close()")
	}
}

func TestCamera_ReadFrame_NotOpened(t *testing.T) {
	cam := NewCamera(0)

	_, err := cam.ReadFrame()
	if err == nil {
		t.Error("ReadFrame() should return error when camera is not open")
	}
}

func TestCamera_Close_NotOpened(t *testing.T) {
	cam := NewCamera(0)

	// Close on not opened camera should not panic and return nil
	err := cam.Close()
	if err != nil {
		t.Errorf("Close() on not opened camera should return nil, got: %v", err)
	}
}
