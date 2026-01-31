package capture

import (
	"image"
	"sync"

	"gocv.io/x/gocv"
)

// MotionDetector detects motion between consecutive video frames
// using frame differencing with Gaussian blur for noise reduction.
type MotionDetector struct {
	threshold   float64
	prevGray    gocv.Mat
	initialized bool
	mu          sync.Mutex
}

// Motion detection constants
const (
	// GaussianBlurSize is the kernel size for Gaussian blur (21x21)
	GaussianBlurSize = 21
	// DiffThreshold is the binary threshold for difference detection
	DiffThreshold = 25
)

// NewMotionDetector creates a new MotionDetector with the given threshold.
// The threshold is the percentage of pixels that must change to detect motion.
// For example, a threshold of 1.0 means 1% of pixels must change.
func NewMotionDetector(threshold float64) *MotionDetector {
	return &MotionDetector{
		threshold:   threshold,
		prevGray:    gocv.NewMat(),
		initialized: false,
	}
}

// Detect analyzes a frame for motion compared to the previous frame.
// Returns whether motion was detected and the percentage of pixels that changed.
//
// Algorithm:
// 1. Convert frame to grayscale
// 2. Apply Gaussian blur (21x21) to reduce noise
// 3. If first frame, store as baseline and return false
// 4. Calculate absolute difference with previous frame
// 5. Threshold the difference (threshold=25)
// 6. Count non-zero pixels / total pixels = changePercent
// 7. Return changePercent > threshold
func (m *MotionDetector) Detect(frame *gocv.Mat) (bool, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if frame == nil || frame.Empty() {
		return false, 0
	}

	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()

	if frame.Channels() > 1 {
		gocv.CvtColor(*frame, &gray, gocv.ColorBGRToGray)
	} else {
		frame.CopyTo(&gray)
	}

	// Apply Gaussian blur to reduce noise
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Point{X: GaussianBlurSize, Y: GaussianBlurSize}, 0, 0, gocv.BorderDefault)

	// If first frame, store as baseline
	if !m.initialized {
		blurred.CopyTo(&m.prevGray)
		m.initialized = true
		return false, 0
	}

	// Calculate absolute difference
	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(blurred, m.prevGray, &diff)

	// Apply binary threshold
	thresh := gocv.NewMat()
	defer thresh.Close()
	gocv.Threshold(diff, &thresh, DiffThreshold, 255, gocv.ThresholdBinary)

	// Count non-zero pixels
	nonZero := gocv.CountNonZero(thresh)
	totalPixels := thresh.Rows() * thresh.Cols()

	// Calculate change percentage
	changePercent := float64(nonZero) / float64(totalPixels) * 100.0

	// Update previous frame
	blurred.CopyTo(&m.prevGray)

	// Return detection result
	return changePercent > m.threshold, changePercent
}

// Reset clears the motion detector state, allowing it to be reused
// with a new baseline frame.
func (m *MotionDetector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.prevGray.Empty() {
		m.prevGray.Close()
		m.prevGray = gocv.NewMat()
	}
	m.initialized = false
}

// Close releases resources used by the motion detector.
func (m *MotionDetector) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.prevGray.Empty() {
		m.prevGray.Close()
		m.prevGray = gocv.NewMat()
	}
	m.initialized = false
}

// SetThreshold sets the motion detection threshold.
// The threshold is the percentage of pixels that must change to detect motion.
// Values less than or equal to 0 are ignored.
func (m *MotionDetector) SetThreshold(threshold float64) {
	if threshold <= 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.threshold = threshold
}
