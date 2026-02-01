package gesture

import (
	"encoding/json"
	"math"
	"testing"
)

func TestTrainer_TrainStatic(t *testing.T) {
	trainer := NewTrainer()

	samples := []json.RawMessage{
		json.RawMessage(`{"type": "static", "landmarks": [{"x": 0.5, "y": 0.5, "z": 0}], "timestamp": 1000}`),
		json.RawMessage(`{"type": "static", "landmarks": [{"x": 0.6, "y": 0.4, "z": 0}], "timestamp": 2000}`),
	}

	result, err := trainer.TrainStatic(samples)
	if err != nil {
		t.Fatalf("TrainStatic() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 landmark, got %d", len(result))
	}

	// Average should be (0.55, 0.45, 0)
	if !floatEqual(result[0].X, 0.55) || !floatEqual(result[0].Y, 0.45) {
		t.Errorf("wrong average: got (%f, %f), expected (0.55, 0.45)", result[0].X, result[0].Y)
	}
}

func TestTrainer_TrainStatic_MultipleLandmarks(t *testing.T) {
	trainer := NewTrainer()

	samples := []json.RawMessage{
		json.RawMessage(`{"type": "static", "landmarks": [{"x": 0.1, "y": 0.1, "z": 0.1}, {"x": 0.2, "y": 0.2, "z": 0.2}], "timestamp": 1000}`),
		json.RawMessage(`{"type": "static", "landmarks": [{"x": 0.3, "y": 0.3, "z": 0.3}, {"x": 0.4, "y": 0.4, "z": 0.4}], "timestamp": 2000}`),
	}

	result, err := trainer.TrainStatic(samples)
	if err != nil {
		t.Fatalf("TrainStatic() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 landmarks, got %d", len(result))
	}

	// First landmark average: (0.2, 0.2, 0.2)
	if !floatEqual(result[0].X, 0.2) || !floatEqual(result[0].Y, 0.2) || !floatEqual(result[0].Z, 0.2) {
		t.Errorf("wrong first average: got (%f, %f, %f)", result[0].X, result[0].Y, result[0].Z)
	}

	// Second landmark average: (0.3, 0.3, 0.3)
	if !floatEqual(result[1].X, 0.3) || !floatEqual(result[1].Y, 0.3) || !floatEqual(result[1].Z, 0.3) {
		t.Errorf("wrong second average: got (%f, %f, %f)", result[1].X, result[1].Y, result[1].Z)
	}
}

func TestTrainer_TrainStatic_EmptySamples(t *testing.T) {
	trainer := NewTrainer()

	_, err := trainer.TrainStatic([]json.RawMessage{})
	if err == nil {
		t.Error("expected error for empty samples")
	}
}

func TestTrainer_TrainStatic_InvalidJSON(t *testing.T) {
	trainer := NewTrainer()

	samples := []json.RawMessage{
		json.RawMessage(`{invalid json}`),
	}

	_, err := trainer.TrainStatic(samples)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTrainer_TrainDynamic(t *testing.T) {
	trainer := NewTrainer()

	samples := []json.RawMessage{
		json.RawMessage(`{"type": "dynamic", "path": [{"x": 0, "y": 0, "timestamp": 0}, {"x": 1, "y": 1, "timestamp": 100}], "timestamp": 1000}`),
		json.RawMessage(`{"type": "dynamic", "path": [{"x": 0, "y": 0, "timestamp": 0}, {"x": 1, "y": 1, "timestamp": 100}], "timestamp": 2000}`),
	}

	result, err := trainer.TrainDynamic(samples)
	if err != nil {
		t.Fatalf("TrainDynamic() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 path points, got %d", len(result))
	}

	// Both samples are identical, so average should match
	if !floatEqual(result[0].X, 0) || !floatEqual(result[0].Y, 0) {
		t.Errorf("wrong first point: got (%f, %f)", result[0].X, result[0].Y)
	}

	if !floatEqual(result[1].X, 1) || !floatEqual(result[1].Y, 1) {
		t.Errorf("wrong second point: got (%f, %f)", result[1].X, result[1].Y)
	}
}

func TestTrainer_TrainDynamic_DifferentLengths(t *testing.T) {
	trainer := NewTrainer()

	// First path has 3 points, second has 5 points
	samples := []json.RawMessage{
		json.RawMessage(`{"type": "dynamic", "path": [{"x": 0, "y": 0, "timestamp": 0}, {"x": 0.5, "y": 0.5, "timestamp": 50}, {"x": 1, "y": 1, "timestamp": 100}], "timestamp": 1000}`),
		json.RawMessage(`{"type": "dynamic", "path": [{"x": 0, "y": 0, "timestamp": 0}, {"x": 0.25, "y": 0.25, "timestamp": 25}, {"x": 0.5, "y": 0.5, "timestamp": 50}, {"x": 0.75, "y": 0.75, "timestamp": 75}, {"x": 1, "y": 1, "timestamp": 100}], "timestamp": 2000}`),
	}

	result, err := trainer.TrainDynamic(samples)
	if err != nil {
		t.Fatalf("TrainDynamic() error = %v", err)
	}

	// Result should have 3 points (length of first sample)
	if len(result) != 3 {
		t.Fatalf("expected 3 path points, got %d", len(result))
	}

	// First and last points should still be at start and end
	if !floatEqual(result[0].X, 0) || !floatEqual(result[0].Y, 0) {
		t.Errorf("wrong first point: got (%f, %f)", result[0].X, result[0].Y)
	}

	if !floatEqual(result[2].X, 1) || !floatEqual(result[2].Y, 1) {
		t.Errorf("wrong last point: got (%f, %f)", result[2].X, result[2].Y)
	}
}

func TestTrainer_TrainDynamic_EmptySamples(t *testing.T) {
	trainer := NewTrainer()

	_, err := trainer.TrainDynamic([]json.RawMessage{})
	if err == nil {
		t.Error("expected error for empty samples")
	}
}

func TestTrainer_TrainDynamic_InsufficientPoints(t *testing.T) {
	trainer := NewTrainer()

	samples := []json.RawMessage{
		json.RawMessage(`{"type": "dynamic", "path": [{"x": 0, "y": 0, "timestamp": 0}], "timestamp": 1000}`),
	}

	_, err := trainer.TrainDynamic(samples)
	if err == nil {
		t.Error("expected error for insufficient path points")
	}
}

func TestResamplePath(t *testing.T) {
	// Test resampling a 3-point path to 5 points
	path := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 1, Timestamp: 100},
		{X: 2, Y: 0, Timestamp: 200},
	}

	result := resamplePath(path, 5)

	if len(result) != 5 {
		t.Fatalf("expected 5 points, got %d", len(result))
	}

	// First point should be (0, 0)
	if !floatEqual(result[0].X, 0) || !floatEqual(result[0].Y, 0) {
		t.Errorf("wrong first point: got (%f, %f)", result[0].X, result[0].Y)
	}

	// Last point should be (2, 0)
	if !floatEqual(result[4].X, 2) || !floatEqual(result[4].Y, 0) {
		t.Errorf("wrong last point: got (%f, %f)", result[4].X, result[4].Y)
	}

	// Middle point should be (1, 1)
	if !floatEqual(result[2].X, 1) || !floatEqual(result[2].Y, 1) {
		t.Errorf("wrong middle point: got (%f, %f)", result[2].X, result[2].Y)
	}
}

func TestResamplePath_SameLength(t *testing.T) {
	path := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 1, Timestamp: 100},
	}

	result := resamplePath(path, 2)

	if len(result) != 2 {
		t.Fatalf("expected 2 points, got %d", len(result))
	}

	if !floatEqual(result[0].X, 0) || !floatEqual(result[1].X, 1) {
		t.Errorf("unexpected result: start=(%f, %f), end=(%f, %f)",
			result[0].X, result[0].Y, result[1].X, result[1].Y)
	}
}

func TestResamplePath_Empty(t *testing.T) {
	result := resamplePath([]PathPoint{}, 5)
	if result != nil {
		t.Error("expected nil for empty path")
	}
}

func TestResamplePath_SinglePoint(t *testing.T) {
	path := []PathPoint{{X: 1, Y: 2, Timestamp: 100}}

	result := resamplePath(path, 5)

	if len(result) != 1 {
		t.Fatalf("expected 1 point, got %d", len(result))
	}

	if !floatEqual(result[0].X, 1) || !floatEqual(result[0].Y, 2) {
		t.Errorf("unexpected point: (%f, %f)", result[0].X, result[0].Y)
	}
}

// floatEqual checks if two floats are approximately equal.
func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
