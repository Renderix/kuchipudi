package detector

import "gocv.io/x/gocv"

// Detector defines the interface for hand detection implementations.
type Detector interface {
	// Detect analyzes a video frame and returns detected hand landmarks.
	// Returns an empty slice if no hands are detected.
	Detect(frame *gocv.Mat) ([]HandLandmarks, error)

	// Close releases any resources held by the detector.
	Close() error
}

// Config holds configuration options for hand detection.
type Config struct {
	// MaxHands is the maximum number of hands to detect (default: 2).
	MaxHands int

	// MinConfidence is the minimum detection confidence threshold (0.0-1.0).
	MinConfidence float64

	// MinTrackingConf is the minimum tracking confidence threshold (0.0-1.0).
	MinTrackingConf float64
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() Config {
	return Config{
		MaxHands:        2,
		MinConfidence:   0.5,
		MinTrackingConf: 0.5,
	}
}
