package gesture

import (
	"math"
	"sort"
)

// DTWDistance calculates Dynamic Time Warping distance between two paths.
// Returns infinity if either path is empty.
// The distance is normalized by the maximum path length.
func DTWDistance(path1, path2 []PathPoint) float64 {
	n := len(path1)
	m := len(path2)

	// Handle empty paths
	if n == 0 || m == 0 {
		return math.Inf(1)
	}

	// Create (n+1) x (m+1) cost matrix initialized to infinity
	dtw := make([][]float64, n+1)
	for i := range dtw {
		dtw[i] = make([]float64, m+1)
		for j := range dtw[i] {
			dtw[i][j] = math.Inf(1)
		}
	}

	// Set dtw[0][0] = 0
	dtw[0][0] = 0

	// Fill in the cost matrix
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			// Cost is the distance between current points plus minimum of three neighbors
			cost := pointDistance(path1[i-1], path2[j-1])
			dtw[i][j] = cost + min3(dtw[i-1][j], dtw[i][j-1], dtw[i-1][j-1])
		}
	}

	// Return normalized distance
	return dtw[n][m] / float64(max(n, m))
}

// pointDistance calculates the Euclidean distance between two PathPoints.
func pointDistance(a, b PathPoint) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// min3 returns the minimum of three float64 values.
func min3(a, b, c float64) float64 {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// max returns the maximum of two int values.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DynamicMatcher matches dynamic gestures against registered templates using DTW.
type DynamicMatcher struct {
	templates []*Template
}

// NewDynamicMatcher creates a new DynamicMatcher instance.
func NewDynamicMatcher() *DynamicMatcher {
	return &DynamicMatcher{
		templates: make([]*Template, 0),
	}
}

// AddTemplate adds a gesture template to the matcher.
func (m *DynamicMatcher) AddTemplate(t *Template) {
	if t == nil {
		return
	}
	m.templates = append(m.templates, t)
}

// RemoveTemplate removes a template by its ID.
func (m *DynamicMatcher) RemoveTemplate(id string) {
	for i, t := range m.templates {
		if t.ID == id {
			// Remove element by shifting
			m.templates = append(m.templates[:i], m.templates[i+1:]...)
			return
		}
	}
}

// Match finds matching templates for the given path.
// Returns matches sorted by score in descending order (best matches first).
func (m *DynamicMatcher) Match(path []PathPoint) []Match {
	if len(path) == 0 {
		return nil
	}

	// Normalize input path
	normalizedInput := normalizePath(path)
	if len(normalizedInput) == 0 {
		return nil
	}

	var matches []Match

	for _, template := range m.templates {
		// Skip non-dynamic templates
		if template.Type != TypeDynamic {
			continue
		}

		// Skip templates with empty paths
		if len(template.Path) == 0 {
			continue
		}

		// Normalize template path
		normalizedTemplate := normalizePath(template.Path)

		// Calculate DTW distance
		distance := DTWDistance(normalizedInput, normalizedTemplate)

		// Skip infinite distances
		if math.IsInf(distance, 1) {
			continue
		}

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

	// Sort matches by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// normalizePath scales the path coordinates to the 0-1 range.
// Timestamps are preserved.
func normalizePath(path []PathPoint) []PathPoint {
	if path == nil {
		return nil
	}

	n := len(path)
	if n == 0 {
		return []PathPoint{}
	}

	// Handle single point case
	if n == 1 {
		return []PathPoint{
			{X: 0, Y: 0, Timestamp: path[0].Timestamp},
		}
	}

	// Find min and max values
	minX, maxX := path[0].X, path[0].X
	minY, maxY := path[0].Y, path[0].Y

	for _, p := range path {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Calculate ranges
	rangeX := maxX - minX
	rangeY := maxY - minY

	// Normalize to 0-1 range
	normalized := make([]PathPoint, n)
	for i, p := range path {
		var normX, normY float64

		if rangeX > 0 {
			normX = (p.X - minX) / rangeX
		}
		if rangeY > 0 {
			normY = (p.Y - minY) / rangeY
		}

		normalized[i] = PathPoint{
			X:         normX,
			Y:         normY,
			Timestamp: p.Timestamp,
		}
	}

	return normalized
}
