package gesture

import (
	"encoding/json"
	"fmt"

	"github.com/ayusman/kuchipudi/internal/detector"
)

// Trainer processes recorded samples into gesture templates.
type Trainer struct{}

// NewTrainer creates a new Trainer instance.
func NewTrainer() *Trainer {
	return &Trainer{}
}

// StaticSample represents a recorded static gesture sample.
type StaticSample struct {
	Type      string             `json:"type"`
	Landmarks []detector.Point3D `json:"landmarks"`
	Timestamp int64              `json:"timestamp"`
}

// DynamicSample represents a recorded dynamic gesture sample.
type DynamicSample struct {
	Type      string      `json:"type"`
	Path      []PathPoint `json:"path"`
	Timestamp int64       `json:"timestamp"`
}

// TrainStatic averages multiple static landmark samples into a single template.
// Returns the averaged landmarks suitable for gesture matching.
func (t *Trainer) TrainStatic(samples []json.RawMessage) ([]detector.Point3D, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("no samples provided")
	}

	// Parse all samples
	var allLandmarks [][]detector.Point3D
	for i, raw := range samples {
		var sample StaticSample
		if err := json.Unmarshal(raw, &sample); err != nil {
			return nil, fmt.Errorf("failed to parse sample %d: %w", i, err)
		}

		if len(sample.Landmarks) == 0 {
			return nil, fmt.Errorf("sample %d has no landmarks", i)
		}

		allLandmarks = append(allLandmarks, sample.Landmarks)
	}

	// Verify all samples have the same number of landmarks
	numPoints := len(allLandmarks[0])
	for i, landmarks := range allLandmarks {
		if len(landmarks) != numPoints {
			return nil, fmt.Errorf("sample %d has %d landmarks, expected %d", i, len(landmarks), numPoints)
		}
	}

	// Average landmarks across all samples
	averaged := make([]detector.Point3D, numPoints)
	n := float64(len(allLandmarks))

	for i := 0; i < numPoints; i++ {
		var sumX, sumY, sumZ float64
		for _, landmarks := range allLandmarks {
			sumX += landmarks[i].X
			sumY += landmarks[i].Y
			sumZ += landmarks[i].Z
		}
		averaged[i] = detector.Point3D{
			X: sumX / n,
			Y: sumY / n,
			Z: sumZ / n,
		}
	}

	return averaged, nil
}

// TrainDynamic averages multiple dynamic path samples into a single template path.
// Uses resampling to align paths of different lengths before averaging.
func (t *Trainer) TrainDynamic(samples []json.RawMessage) ([]PathPoint, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("no samples provided")
	}

	// Parse all samples
	var allPaths [][]PathPoint
	for i, raw := range samples {
		var sample DynamicSample
		if err := json.Unmarshal(raw, &sample); err != nil {
			return nil, fmt.Errorf("failed to parse sample %d: %w", i, err)
		}

		if len(sample.Path) < 2 {
			return nil, fmt.Errorf("sample %d has insufficient path points", i)
		}

		allPaths = append(allPaths, sample.Path)
	}

	// Use the first path as reference length
	targetLength := len(allPaths[0])

	// Resample all paths to the same length and average
	averaged := make([]PathPoint, targetLength)

	for i := 0; i < targetLength; i++ {
		var sumX, sumY float64
		var refTimestamp int64

		for pathIdx, path := range allPaths {
			// Resample path to match target length
			resampled := resamplePath(path, targetLength)

			sumX += resampled[i].X
			sumY += resampled[i].Y

			// Use timestamp from first path as reference
			if pathIdx == 0 {
				refTimestamp = resampled[i].Timestamp
			}
		}

		n := float64(len(allPaths))
		averaged[i] = PathPoint{
			X:         sumX / n,
			Y:         sumY / n,
			Timestamp: refTimestamp,
		}
	}

	return averaged, nil
}

// resamplePath resamples a path to have exactly targetLength points.
// Uses linear interpolation for smooth resampling.
func resamplePath(path []PathPoint, targetLength int) []PathPoint {
	if len(path) == 0 {
		return nil
	}

	if len(path) == 1 || targetLength <= 1 {
		return []PathPoint{path[0]}
	}

	result := make([]PathPoint, targetLength)

	for i := 0; i < targetLength; i++ {
		// Map index i to a position in the original path
		t := float64(i) / float64(targetLength-1)
		pos := t * float64(len(path)-1)

		// Get the two surrounding points for interpolation
		idx := int(pos)
		if idx >= len(path)-1 {
			idx = len(path) - 2
		}

		// Calculate interpolation factor
		frac := pos - float64(idx)

		p1 := path[idx]
		p2 := path[idx+1]

		// Linear interpolation
		result[i] = PathPoint{
			X:         p1.X + frac*(p2.X-p1.X),
			Y:         p1.Y + frac*(p2.Y-p1.Y),
			Timestamp: p1.Timestamp + int64(frac*float64(p2.Timestamp-p1.Timestamp)),
		}
	}

	return result
}
