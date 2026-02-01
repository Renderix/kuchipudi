// Package capture provides camera capture functionality using GoCV (OpenCV).
package capture

import (
	"errors"
	"image"
	"sync"

	"gocv.io/x/gocv"
)

// Default camera settings
const (
	DefaultFPS    = 5
	DefaultWidth  = 640
	DefaultHeight = 480
)

// ErrCameraNotOpen is returned when trying to read from a camera that is not open.
var ErrCameraNotOpen = errors.New("camera is not open")

// Frame represents a captured video frame with metadata.
type Frame struct {
	Image     image.Image
	Timestamp int64
	Width     int
	Height    int
}

// Camera defines the interface for camera capture implementations.
type Camera interface {
	Open() error
	Close() error
	ReadFrame() (*gocv.Mat, error)
	SetFPS(fps int)
	FPS() int
	IsOpen() bool
}

// cameraImpl manages video capture from a camera device using GoCV.
type cameraImpl struct {
	deviceID int
	capture  *gocv.VideoCapture
	mu       sync.Mutex
	running  bool
	fps      int
}

// NewCamera creates a new Camera with the given device ID.
// The default FPS is 5 for performance reasons.
func NewCamera(deviceID int) Camera {
	return &cameraImpl{
		deviceID: deviceID,
		fps:      DefaultFPS,
		running:  false,
		capture:  nil,
	}
}

// Open opens the camera for capturing frames.
// It sets the resolution to 640x480 for performance.
func (c *cameraImpl) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	capture, err := gocv.OpenVideoCapture(c.deviceID)
	if err != nil {
		return err
	}

	// Set resolution for performance
	capture.Set(gocv.VideoCaptureFrameWidth, DefaultWidth)
	capture.Set(gocv.VideoCaptureFrameHeight, DefaultHeight)
	capture.Set(gocv.VideoCaptureFPS, float64(c.fps))

	c.capture = capture
	c.running = true

	return nil
}

// Close closes the camera and releases resources.
func (c *cameraImpl) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running || c.capture == nil {
		c.running = false
		return nil
	}

	err := c.capture.Close()
	c.capture = nil
	c.running = false

	return err
}

// ReadFrame reads a single frame from the camera.
// The caller is responsible for closing the returned Mat.
func (c *cameraImpl) ReadFrame() (*gocv.Mat, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running || c.capture == nil {
		return nil, ErrCameraNotOpen
	}

	mat := gocv.NewMat()
	if ok := c.capture.Read(&mat); !ok {
		mat.Close()
		return nil, errors.New("failed to read frame from camera")
	}

	if mat.Empty() {
		mat.Close()
		return nil, errors.New("captured frame is empty")
	}

	return &mat, nil
}

// SetFPS sets the frames per second for capture.
// Values less than or equal to 0 are ignored.
func (c *cameraImpl) SetFPS(fps int) {
	if fps <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.fps = fps

	if c.capture != nil {
		c.capture.Set(gocv.VideoCaptureFPS, float64(fps))
	}
}

// FPS returns the current frames per second setting.
func (c *cameraImpl) FPS() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.fps
}

// IsOpen returns true if the camera is currently open and running.
func (c *cameraImpl) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.running
}
