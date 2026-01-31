package detector

import (
	"errors"
	"math"
	"testing"
)

const epsilon = 1e-9

func TestHandLandmarks_Normalize(t *testing.T) {
	t.Run("wrist at origin after normalization", func(t *testing.T) {
		// Create a hand with wrist at non-zero position
		hand := HandLandmarks{
			Handedness: "Right",
			Score:      0.9,
		}

		// Set wrist at arbitrary position
		hand.Points[Wrist] = Point3D{X: 100.0, Y: 200.0, Z: 50.0}
		// Set middle MCP relative to wrist (distance of 50 units)
		hand.Points[MiddleMCP] = Point3D{X: 130.0, Y: 240.0, Z: 50.0}

		// Fill other landmarks with some values
		for i := 1; i < NumLandmarks; i++ {
			if i != MiddleMCP {
				hand.Points[i] = Point3D{
					X: 100.0 + float64(i)*10.0,
					Y: 200.0 + float64(i)*5.0,
					Z: 50.0 + float64(i)*2.0,
				}
			}
		}

		normalized := hand.Normalize()

		// Verify wrist is at origin
		if math.Abs(normalized.Points[Wrist].X) > epsilon {
			t.Errorf("expected wrist X to be 0, got %f", normalized.Points[Wrist].X)
		}
		if math.Abs(normalized.Points[Wrist].Y) > epsilon {
			t.Errorf("expected wrist Y to be 0, got %f", normalized.Points[Wrist].Y)
		}
		if math.Abs(normalized.Points[Wrist].Z) > epsilon {
			t.Errorf("expected wrist Z to be 0, got %f", normalized.Points[Wrist].Z)
		}

		// Verify handedness and score are preserved
		if normalized.Handedness != hand.Handedness {
			t.Errorf("expected handedness %s, got %s", hand.Handedness, normalized.Handedness)
		}
		if normalized.Score != hand.Score {
			t.Errorf("expected score %f, got %f", hand.Score, normalized.Score)
		}
	})

	t.Run("distance from wrist to middle MCP is 1.0", func(t *testing.T) {
		hand := HandLandmarks{}

		// Set wrist and middle MCP with known distance
		hand.Points[Wrist] = Point3D{X: 10.0, Y: 20.0, Z: 5.0}
		hand.Points[MiddleMCP] = Point3D{X: 13.0, Y: 24.0, Z: 5.0} // distance = 5.0

		// Fill other landmarks
		for i := 1; i < NumLandmarks; i++ {
			if i != MiddleMCP {
				hand.Points[i] = Point3D{
					X: 10.0 + float64(i),
					Y: 20.0 + float64(i),
					Z: 5.0,
				}
			}
		}

		normalized := hand.Normalize()

		// Calculate distance from wrist (origin) to middle MCP
		middleMCP := normalized.Points[MiddleMCP]
		distance := math.Sqrt(middleMCP.X*middleMCP.X + middleMCP.Y*middleMCP.Y + middleMCP.Z*middleMCP.Z)

		if math.Abs(distance-1.0) > epsilon {
			t.Errorf("expected distance from wrist to middle MCP to be 1.0, got %f", distance)
		}
	})

	t.Run("nil hand returns nil", func(t *testing.T) {
		var hand *HandLandmarks
		normalized := hand.Normalize()

		if normalized != nil {
			t.Error("expected nil result for nil input")
		}
	})

	t.Run("zero scale returns translated only", func(t *testing.T) {
		hand := HandLandmarks{}

		// Set wrist and middle MCP at same position (zero scale)
		hand.Points[Wrist] = Point3D{X: 10.0, Y: 20.0, Z: 5.0}
		hand.Points[MiddleMCP] = Point3D{X: 10.0, Y: 20.0, Z: 5.0}

		normalized := hand.Normalize()

		// Wrist should still be at origin
		if math.Abs(normalized.Points[Wrist].X) > epsilon {
			t.Errorf("expected wrist X to be 0, got %f", normalized.Points[Wrist].X)
		}
	})
}

func TestMockDetector(t *testing.T) {
	t.Run("returns empty hands by default", func(t *testing.T) {
		mock := NewMockDetector()

		hands, err := mock.Detect(nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if hands != nil {
			t.Errorf("expected nil hands, got %v", hands)
		}
	})

	t.Run("returns configured hands", func(t *testing.T) {
		mock := NewMockDetector()

		expectedHands := []HandLandmarks{
			ThumbsUpLandmarks(),
			OpenPalmLandmarks(),
		}
		mock.SetHands(expectedHands)

		hands, err := mock.Detect(nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(hands) != 2 {
			t.Errorf("expected 2 hands, got %d", len(hands))
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		mock := NewMockDetector()

		expectedErr := errors.New("detection failed")
		mock.SetError(expectedErr)

		hands, err := mock.Detect(nil)

		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		if hands != nil {
			t.Errorf("expected nil hands when error is set, got %v", hands)
		}
	})

	t.Run("Close returns nil", func(t *testing.T) {
		mock := NewMockDetector()

		err := mock.Close()

		if err != nil {
			t.Errorf("expected Close to return nil, got %v", err)
		}
	})

	t.Run("implements Detector interface", func(t *testing.T) {
		var _ Detector = (*MockDetector)(nil)
	})
}

func TestThumbsUpLandmarks(t *testing.T) {
	landmarks := ThumbsUpLandmarks()

	t.Run("has correct handedness and score", func(t *testing.T) {
		if landmarks.Handedness != "Right" {
			t.Errorf("expected handedness Right, got %s", landmarks.Handedness)
		}
		if landmarks.Score < 0.9 {
			t.Errorf("expected score >= 0.9, got %f", landmarks.Score)
		}
	})

	t.Run("thumb is extended upward", func(t *testing.T) {
		// Thumb tip should be above (lower Y) than thumb MCP
		if landmarks.Points[ThumbTip].Y >= landmarks.Points[ThumbMCP].Y {
			t.Error("thumb tip should be above thumb MCP (lower Y value)")
		}

		// Thumb tip should be above thumb IP
		if landmarks.Points[ThumbTip].Y >= landmarks.Points[ThumbIP].Y {
			t.Error("thumb tip should be above thumb IP (lower Y value)")
		}
	})

	t.Run("other fingers are curled", func(t *testing.T) {
		// For curled fingers, the tip should be close to or below the MCP in Y
		// and generally curled back toward the palm

		// Index finger
		indexExtension := landmarks.Points[IndexMCP].Y - landmarks.Points[IndexTip].Y
		if indexExtension > 0.15 {
			t.Errorf("index finger appears extended (extension: %f), should be curled", indexExtension)
		}

		// Middle finger
		middleExtension := landmarks.Points[MiddleMCP].Y - landmarks.Points[MiddleTip].Y
		if middleExtension > 0.15 {
			t.Errorf("middle finger appears extended (extension: %f), should be curled", middleExtension)
		}

		// Ring finger
		ringExtension := landmarks.Points[RingMCP].Y - landmarks.Points[RingTip].Y
		if ringExtension > 0.15 {
			t.Errorf("ring finger appears extended (extension: %f), should be curled", ringExtension)
		}

		// Pinky finger
		pinkyExtension := landmarks.Points[PinkyMCP].Y - landmarks.Points[PinkyTip].Y
		if pinkyExtension > 0.15 {
			t.Errorf("pinky finger appears extended (extension: %f), should be curled", pinkyExtension)
		}
	})
}

func TestOpenPalmLandmarks(t *testing.T) {
	landmarks := OpenPalmLandmarks()

	t.Run("has correct handedness and score", func(t *testing.T) {
		if landmarks.Handedness != "Right" {
			t.Errorf("expected handedness Right, got %s", landmarks.Handedness)
		}
		if landmarks.Score < 0.9 {
			t.Errorf("expected score >= 0.9, got %f", landmarks.Score)
		}
	})

	t.Run("all fingers are extended", func(t *testing.T) {
		// For extended fingers, the tip should be significantly above (lower Y) the MCP
		minExtension := 0.2 // minimum expected extension

		// Index finger
		indexExtension := landmarks.Points[IndexMCP].Y - landmarks.Points[IndexTip].Y
		if indexExtension < minExtension {
			t.Errorf("index finger not extended enough (extension: %f), expected >= %f", indexExtension, minExtension)
		}

		// Middle finger
		middleExtension := landmarks.Points[MiddleMCP].Y - landmarks.Points[MiddleTip].Y
		if middleExtension < minExtension {
			t.Errorf("middle finger not extended enough (extension: %f), expected >= %f", middleExtension, minExtension)
		}

		// Ring finger
		ringExtension := landmarks.Points[RingMCP].Y - landmarks.Points[RingTip].Y
		if ringExtension < minExtension {
			t.Errorf("ring finger not extended enough (extension: %f), expected >= %f", ringExtension, minExtension)
		}

		// Pinky finger
		pinkyExtension := landmarks.Points[PinkyMCP].Y - landmarks.Points[PinkyTip].Y
		if pinkyExtension < minExtension {
			t.Errorf("pinky finger not extended enough (extension: %f), expected >= %f", pinkyExtension, minExtension)
		}
	})

	t.Run("thumb is extended to the side", func(t *testing.T) {
		// Thumb should be extended away from the palm (higher X for right hand)
		if landmarks.Points[ThumbTip].X <= landmarks.Points[ThumbMCP].X {
			t.Error("thumb tip should be to the right of thumb MCP (extended outward)")
		}
	})

	t.Run("fingers are properly ordered left to right", func(t *testing.T) {
		// For a right hand palm facing forward, fingers should be ordered
		// from left to right: pinky, ring, middle, index, thumb
		if landmarks.Points[PinkyMCP].X >= landmarks.Points[RingMCP].X {
			t.Error("pinky should be to the left of ring finger")
		}
		if landmarks.Points[RingMCP].X >= landmarks.Points[MiddleMCP].X {
			t.Error("ring should be to the left of middle finger")
		}
		if landmarks.Points[MiddleMCP].X >= landmarks.Points[IndexMCP].X {
			t.Error("middle should be to the left of index finger")
		}
	})
}
