package capture

import (
	"fmt"
	"sync"

	"gocv.io/x/gocv"
)

// MockCamera plays back pre-recorded frames for testing
type MockCamera struct {
	frames  []*gocv.Mat
	index   int
	loop    bool
	mu      sync.Mutex
	running bool
}

func NewMockCamera(frames []*gocv.Mat, loop bool) *MockCamera {
	return &MockCamera{
		frames: frames,
		loop:   loop,
	}
}

func (c *MockCamera) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = true
	c.index = 0
	return nil
}

func (c *MockCamera) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	return nil
}

func (c *MockCamera) ReadFrame() (*gocv.Mat, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil, fmt.Errorf("camera not open")
	}

	if len(c.frames) == 0 {
		return nil, fmt.Errorf("no frames available")
	}

	if c.index >= len(c.frames) {
		if c.loop {
			c.index = 0
		} else {
			return nil, fmt.Errorf("no more frames")
		}
	}

	// Clone the frame so the original isn't modified
	frame := c.frames[c.index].Clone()
	c.index++

	return &frame, nil
}

func (c *MockCamera) SetFPS(fps int) {}
func (c *MockCamera) FPS() int       { return 15 }
func (c *MockCamera) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// SetFrames replaces the frame sequence
func (c *MockCamera) SetFrames(frames []*gocv.Mat) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frames = frames
	c.index = 0
}

// Reset restarts playback from the beginning
func (c *MockCamera) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.index = 0
}
