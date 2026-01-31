package gesture

import (
	"testing"

	"github.com/ayusman/kuchipudi/internal/detector"
)

func TestStaticMatcher_Match(t *testing.T) {
	// Create a static matcher
	matcher := NewStaticMatcher()

	// Create a thumbs up template from normalized thumbs up landmarks
	thumbsUp := detector.ThumbsUpLandmarks()
	normalizedThumbsUp := thumbsUp.Normalize()

	template := &Template{
		ID:        "thumbs-up",
		Name:      "Thumbs Up",
		Type:      TypeStatic,
		Landmarks: normalizedThumbsUp.Points[:],
		Tolerance: 0.5, // Generous tolerance for matching
	}
	matcher.AddTemplate(template)

	// Match against a thumbs up input
	inputThumbsUp := detector.ThumbsUpLandmarks()
	matches := matcher.Match(&inputThumbsUp)

	// Should have at least one match
	if len(matches) == 0 {
		t.Fatal("expected at least one match for thumbs up input")
	}

	// The match should be for our thumbs up template
	if matches[0].Template.ID != "thumbs-up" {
		t.Errorf("expected match for 'thumbs-up' template, got %q", matches[0].Template.ID)
	}

	// The score should be high (close to 1.0) for identical gesture
	if matches[0].Score < 0.9 {
		t.Errorf("expected high score (>0.9) for matching gesture, got %f", matches[0].Score)
	}

	// The distance should be very low for identical gesture
	if matches[0].Distance > 0.1 {
		t.Errorf("expected low distance (<0.1) for matching gesture, got %f", matches[0].Distance)
	}
}

func TestStaticMatcher_NoMatch(t *testing.T) {
	// Create a static matcher
	matcher := NewStaticMatcher()

	// Create a thumbs up template from normalized thumbs up landmarks
	thumbsUp := detector.ThumbsUpLandmarks()
	normalizedThumbsUp := thumbsUp.Normalize()

	template := &Template{
		ID:        "thumbs-up",
		Name:      "Thumbs Up",
		Type:      TypeStatic,
		Landmarks: normalizedThumbsUp.Points[:],
		Tolerance: 0.3, // Stricter tolerance
	}
	matcher.AddTemplate(template)

	// Match against an open palm input (different gesture)
	inputOpenPalm := detector.OpenPalmLandmarks()
	matches := matcher.Match(&inputOpenPalm)

	// Should have no matches since open palm is very different from thumbs up
	if len(matches) > 0 {
		// If there are matches, verify the score is low
		for _, match := range matches {
			if match.Score > 0.5 {
				t.Errorf("expected low score (<0.5) for non-matching gesture, got %f", match.Score)
			}
		}
	}
}

func TestStaticMatcher_AddRemoveTemplate(t *testing.T) {
	matcher := NewStaticMatcher()

	// Create templates
	template1 := &Template{
		ID:        "template-1",
		Name:      "Template 1",
		Type:      TypeStatic,
		Landmarks: make([]detector.Point3D, detector.NumLandmarks),
		Tolerance: 0.5,
	}
	template2 := &Template{
		ID:        "template-2",
		Name:      "Template 2",
		Type:      TypeStatic,
		Landmarks: make([]detector.Point3D, detector.NumLandmarks),
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

func TestStaticMatcher_MultipleMatches(t *testing.T) {
	matcher := NewStaticMatcher()

	// Create a thumbs up template
	thumbsUp := detector.ThumbsUpLandmarks()
	normalizedThumbsUp := thumbsUp.Normalize()

	template1 := &Template{
		ID:        "thumbs-up-1",
		Name:      "Thumbs Up Variant 1",
		Type:      TypeStatic,
		Landmarks: normalizedThumbsUp.Points[:],
		Tolerance: 0.5,
	}

	// Create a similar template with slightly different tolerance
	template2 := &Template{
		ID:        "thumbs-up-2",
		Name:      "Thumbs Up Variant 2",
		Type:      TypeStatic,
		Landmarks: normalizedThumbsUp.Points[:],
		Tolerance: 0.8,
	}

	matcher.AddTemplate(template1)
	matcher.AddTemplate(template2)

	// Match against thumbs up input
	inputThumbsUp := detector.ThumbsUpLandmarks()
	matches := matcher.Match(&inputThumbsUp)

	// Should match both templates
	if len(matches) < 2 {
		t.Errorf("expected at least 2 matches, got %d", len(matches))
	}

	// Matches should be sorted by score descending
	for i := 1; i < len(matches); i++ {
		if matches[i].Score > matches[i-1].Score {
			t.Error("matches should be sorted by score descending")
		}
	}
}

func TestStaticMatcher_NilInput(t *testing.T) {
	matcher := NewStaticMatcher()

	// Create a template
	template := &Template{
		ID:        "test",
		Name:      "Test",
		Type:      TypeStatic,
		Landmarks: make([]detector.Point3D, detector.NumLandmarks),
		Tolerance: 0.5,
	}
	matcher.AddTemplate(template)

	// Match with nil input should return empty matches
	matches := matcher.Match(nil)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for nil input, got %d", len(matches))
	}
}

func TestEuclideanDistance(t *testing.T) {
	// Test with identical points
	a := []detector.Point3D{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 1, Z: 1},
	}
	b := []detector.Point3D{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 1, Z: 1},
	}

	dist := euclideanDistance(a, b)
	if dist != 0 {
		t.Errorf("expected distance 0 for identical points, got %f", dist)
	}

	// Test with different points
	c := []detector.Point3D{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	d := []detector.Point3D{
		{X: 0, Y: 0, Z: 0},
		{X: 2, Y: 0, Z: 0},
	}

	dist2 := euclideanDistance(c, d)
	if dist2 != 1.0 {
		t.Errorf("expected distance 1.0, got %f", dist2)
	}

	// Test with empty slices
	dist3 := euclideanDistance(nil, nil)
	if dist3 != 0 {
		t.Errorf("expected distance 0 for empty slices, got %f", dist3)
	}
}
