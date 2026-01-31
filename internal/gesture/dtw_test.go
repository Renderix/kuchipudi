package gesture

import (
	"math"
	"testing"
)

func TestDTW_IdenticalPaths(t *testing.T) {
	// Same path should have distance 0
	path := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 1, Timestamp: 100},
		{X: 2, Y: 2, Timestamp: 200},
	}

	distance := DTWDistance(path, path)

	if distance != 0 {
		t.Errorf("expected distance 0 for identical paths, got %f", distance)
	}
}

func TestDTW_DifferentPaths(t *testing.T) {
	// Different paths should have distance > 0
	path1 := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 0, Timestamp: 100},
		{X: 2, Y: 0, Timestamp: 200},
	}

	path2 := []PathPoint{
		{X: 0, Y: 2, Timestamp: 0},
		{X: 1, Y: 2, Timestamp: 100},
		{X: 2, Y: 2, Timestamp: 200},
	}

	distance := DTWDistance(path1, path2)

	if distance <= 0 {
		t.Errorf("expected distance > 0 for different paths, got %f", distance)
	}
}

func TestDTW_SpeedInvariant(t *testing.T) {
	// Fast and slow versions of same path should match closely
	// The path traces the same trajectory but at different speeds

	// Fast version - fewer points
	fastPath := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 0, Timestamp: 50},
		{X: 2, Y: 0, Timestamp: 100},
	}

	// Slow version - more points covering the same trajectory
	slowPath := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 0.25, Y: 0, Timestamp: 50},
		{X: 0.5, Y: 0, Timestamp: 100},
		{X: 0.75, Y: 0, Timestamp: 150},
		{X: 1, Y: 0, Timestamp: 200},
		{X: 1.25, Y: 0, Timestamp: 250},
		{X: 1.5, Y: 0, Timestamp: 300},
		{X: 1.75, Y: 0, Timestamp: 350},
		{X: 2, Y: 0, Timestamp: 400},
	}

	// Distance should be small since they follow the same trajectory
	distance := DTWDistance(fastPath, slowPath)

	// DTW should handle speed invariance - distance should be relatively small
	if distance > 0.5 {
		t.Errorf("expected low distance for speed-invariant paths, got %f", distance)
	}
}

func TestDTW_EmptyPaths(t *testing.T) {
	// Empty paths should return infinity
	emptyPath := []PathPoint{}
	path := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 1, Timestamp: 100},
	}

	// Both empty
	dist1 := DTWDistance(emptyPath, emptyPath)
	if !math.IsInf(dist1, 1) {
		t.Errorf("expected infinity for empty paths, got %f", dist1)
	}

	// First empty
	dist2 := DTWDistance(emptyPath, path)
	if !math.IsInf(dist2, 1) {
		t.Errorf("expected infinity when first path is empty, got %f", dist2)
	}

	// Second empty
	dist3 := DTWDistance(path, emptyPath)
	if !math.IsInf(dist3, 1) {
		t.Errorf("expected infinity when second path is empty, got %f", dist3)
	}
}

func TestPointDistance(t *testing.T) {
	a := PathPoint{X: 0, Y: 0, Timestamp: 0}
	b := PathPoint{X: 3, Y: 4, Timestamp: 100}

	dist := pointDistance(a, b)

	// Should be 5 (3-4-5 triangle)
	expected := 5.0
	if math.Abs(dist-expected) > 0.0001 {
		t.Errorf("expected distance %f, got %f", expected, dist)
	}
}

func TestMin3(t *testing.T) {
	tests := []struct {
		a, b, c  float64
		expected float64
	}{
		{1, 2, 3, 1},
		{2, 1, 3, 1},
		{3, 2, 1, 1},
		{1, 1, 1, 1},
		{-1, 0, 1, -1},
	}

	for _, tt := range tests {
		result := min3(tt.a, tt.b, tt.c)
		if result != tt.expected {
			t.Errorf("min3(%f, %f, %f) = %f, expected %f", tt.a, tt.b, tt.c, result, tt.expected)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{1, 1, 1},
		{-1, 0, 0},
		{-2, -1, -1},
	}

	for _, tt := range tests {
		result := max(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("max(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestDynamicMatcher_Match(t *testing.T) {
	matcher := NewDynamicMatcher()

	// Create a swipe left template (moving from right to left)
	swipeLeftTemplate := &Template{
		ID:   "swipe-left",
		Name: "Swipe Left",
		Type: TypeDynamic,
		Path: []PathPoint{
			{X: 1, Y: 0.5, Timestamp: 0},
			{X: 0.75, Y: 0.5, Timestamp: 50},
			{X: 0.5, Y: 0.5, Timestamp: 100},
			{X: 0.25, Y: 0.5, Timestamp: 150},
			{X: 0, Y: 0.5, Timestamp: 200},
		},
		Tolerance: 0.5,
	}

	// Create a swipe right template (moving from left to right)
	swipeRightTemplate := &Template{
		ID:   "swipe-right",
		Name: "Swipe Right",
		Type: TypeDynamic,
		Path: []PathPoint{
			{X: 0, Y: 0.5, Timestamp: 0},
			{X: 0.25, Y: 0.5, Timestamp: 50},
			{X: 0.5, Y: 0.5, Timestamp: 100},
			{X: 0.75, Y: 0.5, Timestamp: 150},
			{X: 1, Y: 0.5, Timestamp: 200},
		},
		Tolerance: 0.5,
	}

	matcher.AddTemplate(swipeLeftTemplate)
	matcher.AddTemplate(swipeRightTemplate)

	// Input: swipe left gesture
	inputSwipeLeft := []PathPoint{
		{X: 100, Y: 50, Timestamp: 0},
		{X: 75, Y: 50, Timestamp: 50},
		{X: 50, Y: 50, Timestamp: 100},
		{X: 25, Y: 50, Timestamp: 150},
		{X: 0, Y: 50, Timestamp: 200},
	}

	matches := matcher.Match(inputSwipeLeft)

	// Should have at least one match
	if len(matches) == 0 {
		t.Fatal("expected at least one match for swipe left input")
	}

	// The best match should be swipe-left template
	if matches[0].Template.ID != "swipe-left" {
		t.Errorf("expected best match to be 'swipe-left', got %q", matches[0].Template.ID)
	}

	// Score should be high for matching gesture
	if matches[0].Score < 0.5 {
		t.Errorf("expected score > 0.5 for matching gesture, got %f", matches[0].Score)
	}
}

func TestDynamicMatcher_AddRemoveTemplate(t *testing.T) {
	matcher := NewDynamicMatcher()

	template1 := &Template{
		ID:   "template-1",
		Name: "Template 1",
		Type: TypeDynamic,
		Path: []PathPoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 1, Y: 1, Timestamp: 100},
		},
		Tolerance: 0.5,
	}
	template2 := &Template{
		ID:   "template-2",
		Name: "Template 2",
		Type: TypeDynamic,
		Path: []PathPoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 1, Y: 1, Timestamp: 100},
		},
		Tolerance: 0.5,
	}

	// Add templates
	matcher.AddTemplate(template1)
	matcher.AddTemplate(template2)

	// Verify both templates are added
	if len(matcher.templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(matcher.templates))
	}

	// Remove first template
	matcher.RemoveTemplate("template-1")

	// Verify only second template remains
	if len(matcher.templates) != 1 {
		t.Errorf("expected 1 template after removal, got %d", len(matcher.templates))
	}

	if matcher.templates[0].ID != "template-2" {
		t.Errorf("expected remaining template to be 'template-2', got %q", matcher.templates[0].ID)
	}

	// Remove non-existent template (should not panic)
	matcher.RemoveTemplate("non-existent")
	if len(matcher.templates) != 1 {
		t.Errorf("expected 1 template after removing non-existent, got %d", len(matcher.templates))
	}
}

func TestDynamicMatcher_EmptyInput(t *testing.T) {
	matcher := NewDynamicMatcher()

	template := &Template{
		ID:   "test",
		Name: "Test",
		Type: TypeDynamic,
		Path: []PathPoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 1, Y: 1, Timestamp: 100},
		},
		Tolerance: 0.5,
	}
	matcher.AddTemplate(template)

	// Match with empty input should return empty matches
	matches := matcher.Match(nil)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for nil input, got %d", len(matches))
	}

	matches = matcher.Match([]PathPoint{})
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for empty input, got %d", len(matches))
	}
}

func TestDynamicMatcher_SkipsStaticTemplates(t *testing.T) {
	matcher := NewDynamicMatcher()

	// Add a static template (should be skipped)
	staticTemplate := &Template{
		ID:        "static-template",
		Name:      "Static Template",
		Type:      TypeStatic,
		Landmarks: nil,
		Tolerance: 0.5,
	}
	matcher.AddTemplate(staticTemplate)

	// Add a dynamic template
	dynamicTemplate := &Template{
		ID:   "dynamic-template",
		Name: "Dynamic Template",
		Type: TypeDynamic,
		Path: []PathPoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 1, Y: 1, Timestamp: 100},
		},
		Tolerance: 1.0,
	}
	matcher.AddTemplate(dynamicTemplate)

	// Match should only return dynamic template
	input := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 1, Timestamp: 100},
	}
	matches := matcher.Match(input)

	for _, match := range matches {
		if match.Template.Type == TypeStatic {
			t.Error("expected matcher to skip static templates")
		}
	}
}

func TestNormalizePath(t *testing.T) {
	// Test normalization scales to 0-1 range
	path := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 50, Y: 100, Timestamp: 50},
		{X: 100, Y: 200, Timestamp: 100},
	}

	normalized := normalizePath(path)

	if len(normalized) != len(path) {
		t.Errorf("expected normalized path length %d, got %d", len(path), len(normalized))
	}

	// Check that all values are in 0-1 range
	for i, p := range normalized {
		if p.X < 0 || p.X > 1 {
			t.Errorf("point %d: X=%f is not in 0-1 range", i, p.X)
		}
		if p.Y < 0 || p.Y > 1 {
			t.Errorf("point %d: Y=%f is not in 0-1 range", i, p.Y)
		}
	}

	// Check min/max are 0 and 1
	if normalized[0].X != 0 || normalized[0].Y != 0 {
		t.Errorf("expected first point to be (0, 0), got (%f, %f)", normalized[0].X, normalized[0].Y)
	}
	if normalized[2].X != 1 || normalized[2].Y != 1 {
		t.Errorf("expected last point to be (1, 1), got (%f, %f)", normalized[2].X, normalized[2].Y)
	}
}

func TestNormalizePath_Empty(t *testing.T) {
	// Empty path should return empty
	normalized := normalizePath(nil)
	if normalized != nil {
		t.Errorf("expected nil for nil input, got %v", normalized)
	}

	normalized = normalizePath([]PathPoint{})
	if len(normalized) != 0 {
		t.Errorf("expected empty slice for empty input, got %v", normalized)
	}
}

func TestNormalizePath_SinglePoint(t *testing.T) {
	// Single point should normalize to (0, 0)
	path := []PathPoint{
		{X: 50, Y: 100, Timestamp: 0},
	}

	normalized := normalizePath(path)

	if len(normalized) != 1 {
		t.Fatalf("expected 1 point, got %d", len(normalized))
	}

	// Single point should be at origin
	if normalized[0].X != 0 || normalized[0].Y != 0 {
		t.Errorf("expected (0, 0), got (%f, %f)", normalized[0].X, normalized[0].Y)
	}
}

func TestNormalizePath_PreservesTimestamp(t *testing.T) {
	path := []PathPoint{
		{X: 0, Y: 0, Timestamp: 100},
		{X: 50, Y: 50, Timestamp: 200},
		{X: 100, Y: 100, Timestamp: 300},
	}

	normalized := normalizePath(path)

	for i, p := range normalized {
		if p.Timestamp != path[i].Timestamp {
			t.Errorf("point %d: timestamp %d != original %d", i, p.Timestamp, path[i].Timestamp)
		}
	}
}
