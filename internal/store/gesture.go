package store

import (
	"database/sql"
	"errors"
	"time"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// GestureType represents the type of gesture (static or dynamic).
type GestureType string

const (
	// GestureTypeStatic represents a static hand pose gesture.
	GestureTypeStatic GestureType = "static"
	// GestureTypeDynamic represents a dynamic motion-based gesture.
	GestureTypeDynamic GestureType = "dynamic"
)

// Gesture represents a gesture definition stored in the database.
type Gesture struct {
	ID        string
	Name      string
	Type      GestureType
	Tolerance float64
	Samples   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Landmark represents a single 3D point from the gesture_landmarks table.
type Landmark struct {
	Index int
	X     float64
	Y     float64
	Z     float64
}

// PathPoint represents a point in a gesture path from the gesture_paths table.
type PathPoint struct {
	Sequence    int
	X           float64
	Y           float64
	TimestampMs int64
}

// GestureRepository provides CRUD operations for gestures.
type GestureRepository struct {
	db *sql.DB
}

// Gestures returns the gesture repository for this store.
func (s *Store) Gestures() *GestureRepository {
	return &GestureRepository{db: s.db}
}

// Create inserts a new gesture into the database.
func (r *GestureRepository) Create(g *Gesture) error {
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now

	_, err := r.db.Exec(
		`INSERT INTO gestures (id, name, type, tolerance, samples, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.Name, string(g.Type), g.Tolerance, g.Samples, g.CreatedAt, g.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a gesture by its ID.
func (r *GestureRepository) GetByID(id string) (*Gesture, error) {
	g := &Gesture{}
	var gestureType string

	err := r.db.QueryRow(
		`SELECT id, name, type, tolerance, samples, created_at, updated_at
		 FROM gestures WHERE id = ?`,
		id,
	).Scan(&g.ID, &g.Name, &gestureType, &g.Tolerance, &g.Samples, &g.CreatedAt, &g.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	g.Type = GestureType(gestureType)
	return g, nil
}

// GetByName retrieves a gesture by its name.
func (r *GestureRepository) GetByName(name string) (*Gesture, error) {
	g := &Gesture{}
	var gestureType string

	err := r.db.QueryRow(
		`SELECT id, name, type, tolerance, samples, created_at, updated_at
		 FROM gestures WHERE name = ?`,
		name,
	).Scan(&g.ID, &g.Name, &gestureType, &g.Tolerance, &g.Samples, &g.CreatedAt, &g.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	g.Type = GestureType(gestureType)
	return g, nil
}

// List retrieves all gestures from the database.
func (r *GestureRepository) List() ([]*Gesture, error) {
	rows, err := r.db.Query(
		`SELECT id, name, type, tolerance, samples, created_at, updated_at
		 FROM gestures ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gestures []*Gesture
	for rows.Next() {
		g := &Gesture{}
		var gestureType string

		err := rows.Scan(&g.ID, &g.Name, &gestureType, &g.Tolerance, &g.Samples, &g.CreatedAt, &g.UpdatedAt)
		if err != nil {
			return nil, err
		}

		g.Type = GestureType(gestureType)
		gestures = append(gestures, g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return gestures, nil
}

// Update updates an existing gesture in the database.
func (r *GestureRepository) Update(g *Gesture) error {
	g.UpdatedAt = time.Now()

	result, err := r.db.Exec(
		`UPDATE gestures SET name = ?, type = ?, tolerance = ?, samples = ?, updated_at = ?
		 WHERE id = ?`,
		g.Name, string(g.Type), g.Tolerance, g.Samples, g.UpdatedAt, g.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes a gesture from the database by its ID.
func (r *GestureRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM gestures WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetLandmarks retrieves the normalized landmarks for a static gesture.
// Returns an empty slice if no landmarks are stored (gesture not yet trained).
func (r *GestureRepository) GetLandmarks(gestureID string) ([]Landmark, error) {
	rows, err := r.db.Query(
		`SELECT landmark_index, x, y, z FROM gesture_landmarks
		 WHERE gesture_id = ? ORDER BY landmark_index`,
		gestureID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var landmarks []Landmark
	for rows.Next() {
		var l Landmark
		if err := rows.Scan(&l.Index, &l.X, &l.Y, &l.Z); err != nil {
			return nil, err
		}
		landmarks = append(landmarks, l)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return landmarks, nil
}

// GetPath retrieves the path points for a dynamic gesture.
// Returns an empty slice if no path is stored (gesture not yet trained).
func (r *GestureRepository) GetPath(gestureID string) ([]PathPoint, error) {
	rows, err := r.db.Query(
		`SELECT sequence, x, y, timestamp_ms FROM gesture_paths
		 WHERE gesture_id = ? ORDER BY sequence`,
		gestureID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var path []PathPoint
	for rows.Next() {
		var p PathPoint
		if err := rows.Scan(&p.Sequence, &p.X, &p.Y, &p.TimestampMs); err != nil {
			return nil, err
		}
		path = append(path, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return path, nil
}
