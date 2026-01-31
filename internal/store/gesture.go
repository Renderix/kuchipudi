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
