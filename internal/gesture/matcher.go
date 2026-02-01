// Package gesture provides gesture recognition and matching capabilities.
package gesture

import (
	"math"
	"sort"

	"github.com/ayusman/kuchipudi/internal/detector"
)

// Type represents the type of gesture (static or dynamic).
type Type string

const (
	// TypeStatic represents a static gesture (single hand pose).
	TypeStatic Type = "static"
	// TypeDynamic represents a dynamic gesture (motion over time).
	TypeDynamic Type = "dynamic"
)

// Template represents a gesture template for matching.
type Template struct {
	ID        string             // Unique identifier for the template
	Name      string             // Human-readable name
	Type      Type               // Static or dynamic gesture type
	Landmarks []detector.Point3D // Normalized landmarks for static gestures
	Path      []PathPoint        // Path points for dynamic gestures
	Tolerance float64            // Maximum distance for a match
}

// PathPoint represents a point in a dynamic gesture path.
type PathPoint struct {
	X         float64 // X coordinate
	Y         float64 // Y coordinate
	Timestamp int64   // Timestamp in milliseconds
}

// Match represents a matching result between input and a template.
type Match struct {
	Template *Template // The matched template
	Score    float64   // Match score (0-1, higher is better)
	Distance float64   // Euclidean distance between input and template
}

// StaticMatcher matches static hand gestures against registered templates.
type StaticMatcher struct {
	templates []*Template
	OnMatch   func(id, name string)
}

// NewStaticMatcher creates a new StaticMatcher instance.
func NewStaticMatcher() *StaticMatcher {
	return &StaticMatcher{
		templates: make([]*Template, 0),
	}
}

// AddTemplate adds a gesture template to the matcher.
func (m *StaticMatcher) AddTemplate(t *Template) {
	if t == nil {
		return
	}
	m.templates = append(m.templates, t)
}

// RemoveTemplate removes a template by its ID.
func (m *StaticMatcher) RemoveTemplate(id string) {
	for i, t := range m.templates {
		if t.ID == id {
			// Remove element by shifting
			m.templates = append(m.templates[:i], m.templates[i+1:]...)
			return
		}
	}
}

// Match finds matching templates for the given hand landmarks.
// Returns matches sorted by score in descending order (best matches first).
func (m *StaticMatcher) Match(hand *detector.HandLandmarks) []Match {
	if hand == nil {
		return nil
	}

	// Step 1: Normalize input landmarks
	normalized := hand.Normalize()
	if normalized == nil {
		return nil
	}

	inputLandmarks := normalized.Points[:]

	var matches []Match

	// Step 2-4: For each static template, compute distance and score
	for _, template := range m.templates {
		if template.Type != TypeStatic {
			continue
		}

		// Compute Euclidean distance
		distance := euclideanDistance(inputLandmarks, template.Landmarks)

		// Calculate score: 1.0 / (1.0 + distance)
		score := 1.0 / (1.0 + distance)

		// Only include if distance is within tolerance
		if distance <= template.Tolerance {
			matches = append(matches, Match{
				Template: template,
				Score:    score,
				Distance: distance,
			})
		}
	}

	// Step 5: Sort matches by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// euclideanDistance calculates the total Euclidean distance between two sets of 3D points.
// It sums the distances between corresponding points in the two slices.
func euclideanDistance(a, b []detector.Point3D) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	var totalDist float64
	for i := 0; i < minLen; i++ {
		dx := a[i].X - b[i].X
		dy := a[i].Y - b[i].Y
		dz := a[i].Z - b[i].Z
		totalDist += math.Sqrt(dx*dx + dy*dy + dz*dz)
	}

	return totalDist
}
