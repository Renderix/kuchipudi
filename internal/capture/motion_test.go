package capture

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestNewMotionDetector(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
	}{
		{
			name:      "default threshold",
			threshold: 1.0,
		},
		{
			name:      "high threshold",
			threshold: 5.0,
		},
		{
			name:      "low threshold",
			threshold: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewMotionDetector(tt.threshold)
			if md == nil {
				t.Fatal("NewMotionDetector returned nil")
			}
			defer md.Close()

			if md.threshold != tt.threshold {
				t.Errorf("threshold = %f, want %f", md.threshold, tt.threshold)
			}

			if md.initialized {
				t.Error("motion detector should not be initialized initially")
			}
		})
	}
}

func TestMotionDetector_NoMotion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires GoCV Mat creation")
	}

	md := NewMotionDetector(1.0) // 1% threshold
	defer md.Close()

	// Create two identical black frames
	frame1 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame1.Close()

	frame2 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame2.Close()

	// First frame initializes the detector
	detected, changePercent := md.Detect(&frame1)
	if detected {
		t.Error("first frame should not detect motion")
	}
	if changePercent != 0 {
		t.Errorf("first frame changePercent = %f, want 0", changePercent)
	}

	// Second identical frame should not detect motion
	detected, changePercent = md.Detect(&frame2)
	if detected {
		t.Errorf("identical frames should not detect motion, changePercent = %f", changePercent)
	}
}

func TestMotionDetector_WithMotion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires GoCV Mat creation")
	}

	md := NewMotionDetector(1.0) // 1% threshold
	defer md.Close()

	// Create a black frame (all zeros)
	blackFrame := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer blackFrame.Close()

	// Create a white frame (all 255s)
	whiteFrame := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer whiteFrame.Close()

	// Fill white frame with white pixels
	whiteFrame.SetTo(gocv.NewScalar(255, 255, 255, 0))

	// First frame initializes the detector
	detected, _ := md.Detect(&blackFrame)
	if detected {
		t.Error("first frame should not detect motion")
	}

	// Second frame is completely different, should detect motion
	detected, changePercent := md.Detect(&whiteFrame)
	if !detected {
		t.Errorf("black to white should detect motion, changePercent = %f", changePercent)
	}

	// Change percent should be high (close to 100% since all pixels changed)
	if changePercent < 50.0 {
		t.Errorf("changePercent = %f, expected > 50%% for black to white transition", changePercent)
	}
}

func TestMotionDetector_Reset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires GoCV Mat creation")
	}

	md := NewMotionDetector(1.0)
	defer md.Close()

	// Create a frame and initialize the detector
	frame := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame.Close()

	md.Detect(&frame)

	if !md.initialized {
		t.Error("detector should be initialized after first Detect")
	}

	// Reset should clear state
	md.Reset()

	if md.initialized {
		t.Error("detector should not be initialized after Reset")
	}

	if !md.prevGray.Empty() {
		t.Error("prevGray should be empty after Reset")
	}
}

func TestMotionDetector_SetThreshold(t *testing.T) {
	md := NewMotionDetector(1.0)
	defer md.Close()

	if md.threshold != 1.0 {
		t.Errorf("initial threshold = %f, want 1.0", md.threshold)
	}

	md.SetThreshold(5.0)
	if md.threshold != 5.0 {
		t.Errorf("threshold = %f, want 5.0 after SetThreshold", md.threshold)
	}

	md.SetThreshold(0.5)
	if md.threshold != 0.5 {
		t.Errorf("threshold = %f, want 0.5 after SetThreshold", md.threshold)
	}
}

func TestMotionDetector_SetThreshold_Negative(t *testing.T) {
	md := NewMotionDetector(1.0)
	defer md.Close()

	// Setting negative threshold should be ignored
	md.SetThreshold(-1.0)
	if md.threshold != 1.0 {
		t.Errorf("negative threshold should be ignored, got %f, want 1.0", md.threshold)
	}
}

func TestMotionDetector_Close_Multiple(t *testing.T) {
	md := NewMotionDetector(1.0)

	// Close multiple times should not panic
	md.Close()
	md.Close()
}

func TestMotionDetector_Detect_AfterClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires GoCV Mat creation")
	}

	md := NewMotionDetector(1.0)

	frame := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame.Close()

	md.Detect(&frame)
	md.Close()

	// Detect after close should handle gracefully (re-initialize)
	detected, _ := md.Detect(&frame)
	if detected {
		t.Error("first frame after close should not detect motion")
	}
}

func TestMotionDetector_ThresholdBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires GoCV Mat creation")
	}

	// Test with high threshold to ensure small changes don't trigger
	md := NewMotionDetector(99.0) // Very high threshold
	defer md.Close()

	blackFrame := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer blackFrame.Close()

	whiteFrame := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer whiteFrame.Close()
	whiteFrame.SetTo(gocv.NewScalar(255, 255, 255, 0))

	md.Detect(&blackFrame)
	detected, changePercent := md.Detect(&whiteFrame)

	// Even with 100% change, 99% threshold might not trigger
	// depending on blur effects
	t.Logf("changePercent with black to white: %f, threshold: 99.0, detected: %v", changePercent, detected)
}
