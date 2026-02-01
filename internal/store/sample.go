package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Sample represents a recorded gesture sample stored in the database.
type Sample struct {
	ID          int64           `json:"id"`
	GestureID   string          `json:"gesture_id"`
	SampleIndex int             `json:"sample_index"`
	Data        json.RawMessage `json:"data"`
	CreatedAt   time.Time       `json:"created_at"`
}

// SampleRepository provides CRUD operations for gesture samples.
type SampleRepository struct {
	db *sql.DB
}

// Samples returns the sample repository for this store.
func (s *Store) Samples() *SampleRepository {
	return &SampleRepository{db: s.db}
}

// Create inserts multiple samples for a gesture in a single transaction.
// It also updates the sample count on the gesture.
func (r *SampleRepository) Create(gestureID string, samples []json.RawMessage) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO gesture_samples (gesture_id, sample_index, data) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, data := range samples {
		if _, err := stmt.Exec(gestureID, i, string(data)); err != nil {
			return err
		}
	}

	// Update sample count on the gesture
	_, err = tx.Exec(`UPDATE gestures SET samples = ?, updated_at = ? WHERE id = ?`,
		len(samples), time.Now(), gestureID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetByGestureID retrieves all samples for a given gesture.
func (r *SampleRepository) GetByGestureID(gestureID string) ([]Sample, error) {
	rows, err := r.db.Query(
		`SELECT id, gesture_id, sample_index, data, created_at
		 FROM gesture_samples
		 WHERE gesture_id = ?
		 ORDER BY sample_index`,
		gestureID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []Sample
	for rows.Next() {
		var s Sample
		var data string
		if err := rows.Scan(&s.ID, &s.GestureID, &s.SampleIndex, &data, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.Data = json.RawMessage(data)
		samples = append(samples, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return samples, nil
}

// DeleteByGestureID removes all samples for a given gesture.
func (r *SampleRepository) DeleteByGestureID(gestureID string) error {
	_, err := r.db.Exec(`DELETE FROM gesture_samples WHERE gesture_id = ?`, gestureID)
	return err
}
