package detector

import (
	"gocv.io/x/gocv"
)

// MockDetector is a test implementation of the Detector interface.
// It allows tests to control the detection results.
type MockDetector struct {
	hands []HandLandmarks
	err   error
}

// NewMockDetector creates a new MockDetector instance.
func NewMockDetector() *MockDetector {
	return &MockDetector{}
}

// SetHands sets the hands that will be returned by Detect.
func (m *MockDetector) SetHands(hands []HandLandmarks) {
	m.hands = hands
}

// SetError sets the error that will be returned by Detect.
func (m *MockDetector) SetError(err error) {
	m.err = err
}

// Detect returns the pre-configured hands or error.
func (m *MockDetector) Detect(frame *gocv.Mat) ([]HandLandmarks, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hands, nil
}

// Close is a no-op for the mock detector.
func (m *MockDetector) Close() error {
	return nil
}

// ThumbsUpLandmarks returns a preset HandLandmarks representing a thumbs up gesture.
// The thumb is extended upward while other fingers are curled.
func ThumbsUpLandmarks() HandLandmarks {
	landmarks := HandLandmarks{
		Handedness: "Right",
		Score:      0.95,
	}

	// Wrist at origin
	landmarks.Points[Wrist] = Point3D{X: 0.5, Y: 0.8, Z: 0.0}

	// Thumb extended upward (pointing up, Y decreases going up)
	landmarks.Points[ThumbCMC] = Point3D{X: 0.55, Y: 0.75, Z: 0.0}
	landmarks.Points[ThumbMCP] = Point3D{X: 0.58, Y: 0.65, Z: 0.0}
	landmarks.Points[ThumbIP] = Point3D{X: 0.58, Y: 0.50, Z: 0.0}
	landmarks.Points[ThumbTip] = Point3D{X: 0.58, Y: 0.35, Z: 0.0}

	// Index finger curled (knuckles close together, tip near palm)
	landmarks.Points[IndexMCP] = Point3D{X: 0.55, Y: 0.70, Z: -0.02}
	landmarks.Points[IndexPIP] = Point3D{X: 0.55, Y: 0.68, Z: -0.05}
	landmarks.Points[IndexDIP] = Point3D{X: 0.52, Y: 0.70, Z: -0.04}
	landmarks.Points[IndexTip] = Point3D{X: 0.50, Y: 0.72, Z: -0.02}

	// Middle finger curled
	landmarks.Points[MiddleMCP] = Point3D{X: 0.50, Y: 0.68, Z: -0.02}
	landmarks.Points[MiddlePIP] = Point3D{X: 0.50, Y: 0.66, Z: -0.05}
	landmarks.Points[MiddleDIP] = Point3D{X: 0.47, Y: 0.68, Z: -0.04}
	landmarks.Points[MiddleTip] = Point3D{X: 0.45, Y: 0.70, Z: -0.02}

	// Ring finger curled
	landmarks.Points[RingMCP] = Point3D{X: 0.45, Y: 0.70, Z: -0.02}
	landmarks.Points[RingPIP] = Point3D{X: 0.45, Y: 0.68, Z: -0.05}
	landmarks.Points[RingDIP] = Point3D{X: 0.42, Y: 0.70, Z: -0.04}
	landmarks.Points[RingTip] = Point3D{X: 0.40, Y: 0.72, Z: -0.02}

	// Pinky finger curled
	landmarks.Points[PinkyMCP] = Point3D{X: 0.40, Y: 0.72, Z: -0.02}
	landmarks.Points[PinkyPIP] = Point3D{X: 0.40, Y: 0.70, Z: -0.05}
	landmarks.Points[PinkyDIP] = Point3D{X: 0.37, Y: 0.72, Z: -0.04}
	landmarks.Points[PinkyTip] = Point3D{X: 0.35, Y: 0.74, Z: -0.02}

	return landmarks
}

// OpenPalmLandmarks returns a preset HandLandmarks representing an open palm gesture.
// All fingers are extended outward.
func OpenPalmLandmarks() HandLandmarks {
	landmarks := HandLandmarks{
		Handedness: "Right",
		Score:      0.95,
	}

	// Wrist at base
	landmarks.Points[Wrist] = Point3D{X: 0.5, Y: 0.8, Z: 0.0}

	// Thumb extended to the side
	landmarks.Points[ThumbCMC] = Point3D{X: 0.55, Y: 0.75, Z: 0.02}
	landmarks.Points[ThumbMCP] = Point3D{X: 0.62, Y: 0.70, Z: 0.03}
	landmarks.Points[ThumbIP] = Point3D{X: 0.68, Y: 0.65, Z: 0.03}
	landmarks.Points[ThumbTip] = Point3D{X: 0.73, Y: 0.60, Z: 0.03}

	// Index finger extended upward
	landmarks.Points[IndexMCP] = Point3D{X: 0.55, Y: 0.68, Z: 0.0}
	landmarks.Points[IndexPIP] = Point3D{X: 0.57, Y: 0.55, Z: 0.0}
	landmarks.Points[IndexDIP] = Point3D{X: 0.58, Y: 0.45, Z: 0.0}
	landmarks.Points[IndexTip] = Point3D{X: 0.58, Y: 0.35, Z: 0.0}

	// Middle finger extended upward (slightly longer)
	landmarks.Points[MiddleMCP] = Point3D{X: 0.50, Y: 0.66, Z: 0.0}
	landmarks.Points[MiddlePIP] = Point3D{X: 0.50, Y: 0.52, Z: 0.0}
	landmarks.Points[MiddleDIP] = Point3D{X: 0.50, Y: 0.40, Z: 0.0}
	landmarks.Points[MiddleTip] = Point3D{X: 0.50, Y: 0.28, Z: 0.0}

	// Ring finger extended upward
	landmarks.Points[RingMCP] = Point3D{X: 0.45, Y: 0.68, Z: 0.0}
	landmarks.Points[RingPIP] = Point3D{X: 0.43, Y: 0.55, Z: 0.0}
	landmarks.Points[RingDIP] = Point3D{X: 0.42, Y: 0.45, Z: 0.0}
	landmarks.Points[RingTip] = Point3D{X: 0.42, Y: 0.35, Z: 0.0}

	// Pinky finger extended upward
	landmarks.Points[PinkyMCP] = Point3D{X: 0.40, Y: 0.70, Z: 0.0}
	landmarks.Points[PinkyPIP] = Point3D{X: 0.37, Y: 0.60, Z: 0.0}
	landmarks.Points[PinkyDIP] = Point3D{X: 0.35, Y: 0.50, Z: 0.0}
	landmarks.Points[PinkyTip] = Point3D{X: 0.34, Y: 0.42, Z: 0.0}

	return landmarks
}
