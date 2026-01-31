// Package detector provides hand detection interfaces and types for gesture recognition.
package detector

import "math"

// Hand landmark indices following MediaPipe convention.
// See: https://developers.google.com/mediapipe/solutions/vision/hand_landmarker
const (
	Wrist        = 0
	ThumbCMC     = 1
	ThumbMCP     = 2
	ThumbIP      = 3
	ThumbTip     = 4
	IndexMCP     = 5
	IndexPIP     = 6
	IndexDIP     = 7
	IndexTip     = 8
	MiddleMCP    = 9
	MiddlePIP    = 10
	MiddleDIP    = 11
	MiddleTip    = 12
	RingMCP      = 13
	RingPIP      = 14
	RingDIP      = 15
	RingTip      = 16
	PinkyMCP     = 17
	PinkyPIP     = 18
	PinkyDIP     = 19
	PinkyTip     = 20
	NumLandmarks = 21
)

// Point3D represents a 3D point in space with x, y, z coordinates.
type Point3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// HandLandmarks represents the 21 hand landmarks detected by MediaPipe.
type HandLandmarks struct {
	Points     [NumLandmarks]Point3D `json:"points"`
	Handedness string                `json:"handedness"` // "Left" or "Right"
	Score      float64               `json:"score"`
}

// distance3D calculates the Euclidean distance between two 3D points.
func distance3D(a, b Point3D) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Normalize normalizes the hand landmarks relative to wrist position and hand size.
// The normalized landmarks have the wrist at origin (0,0,0) and are scaled
// so that the distance from wrist to middle finger MCP is 1.0.
// Returns a new HandLandmarks instance with normalized points.
func (h *HandLandmarks) Normalize() *HandLandmarks {
	if h == nil {
		return nil
	}

	normalized := &HandLandmarks{
		Handedness: h.Handedness,
		Score:      h.Score,
	}

	// Get wrist position as the origin
	wrist := h.Points[Wrist]

	// Translate all points relative to wrist
	for i := 0; i < NumLandmarks; i++ {
		normalized.Points[i] = Point3D{
			X: h.Points[i].X - wrist.X,
			Y: h.Points[i].Y - wrist.Y,
			Z: h.Points[i].Z - wrist.Z,
		}
	}

	// Calculate scale factor using distance from wrist to middle finger MCP
	middleMCP := normalized.Points[MiddleMCP]
	scale := distance3D(Point3D{0, 0, 0}, middleMCP)

	// Avoid division by zero
	if scale < 1e-10 {
		return normalized
	}

	// Scale all points
	for i := 0; i < NumLandmarks; i++ {
		normalized.Points[i].X /= scale
		normalized.Points[i].Y /= scale
		normalized.Points[i].Z /= scale
	}

	return normalized
}
