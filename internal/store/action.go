package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Action represents a gesture-to-plugin binding stored in the database.
type Action struct {
	ID         string
	GestureID  string
	PluginName string
	ActionName string
	Config     json.RawMessage
	Enabled    bool
	CreatedAt  time.Time
}

// ActionRepository provides CRUD operations for actions.
type ActionRepository struct {
	db *sql.DB
}

// Actions returns the action repository for this store.
func (s *Store) Actions() *ActionRepository {
	return &ActionRepository{db: s.db}
}

// Create inserts a new action into the database.
func (r *ActionRepository) Create(a *Action) error {
	a.CreatedAt = time.Now()

	config := a.Config
	if config == nil {
		config = json.RawMessage("{}")
	}

	_, err := r.db.Exec(
		`INSERT INTO actions (id, gesture_id, plugin_name, action_name, config, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.GestureID, a.PluginName, a.ActionName, string(config), a.Enabled, a.CreatedAt,
	)
	return err
}

// GetByID retrieves an action by its ID.
func (r *ActionRepository) GetByID(id string) (*Action, error) {
	a := &Action{}
	var config string
	var enabled int

	err := r.db.QueryRow(
		`SELECT id, gesture_id, plugin_name, action_name, config, enabled, created_at
		 FROM actions WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.GestureID, &a.PluginName, &a.ActionName, &config, &enabled, &a.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	a.Config = json.RawMessage(config)
	a.Enabled = enabled != 0
	return a, nil
}

// GetByGestureID retrieves an action by its gesture ID.
// Returns nil, nil if no action is bound to the gesture.
func (r *ActionRepository) GetByGestureID(gestureID string) (*Action, error) {
	a := &Action{}
	var config string
	var enabled int

	err := r.db.QueryRow(
		`SELECT id, gesture_id, plugin_name, action_name, config, enabled, created_at
		 FROM actions WHERE gesture_id = ?`,
		gestureID,
	).Scan(&a.ID, &a.GestureID, &a.PluginName, &a.ActionName, &config, &enabled, &a.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Silent skip - no action bound
		}
		return nil, err
	}

	a.Config = json.RawMessage(config)
	a.Enabled = enabled != 0
	return a, nil
}

// List retrieves all actions from the database.
func (r *ActionRepository) List() ([]*Action, error) {
	rows, err := r.db.Query(
		`SELECT id, gesture_id, plugin_name, action_name, config, enabled, created_at
		 FROM actions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*Action
	for rows.Next() {
		a := &Action{}
		var config string
		var enabled int

		err := rows.Scan(&a.ID, &a.GestureID, &a.PluginName, &a.ActionName, &config, &enabled, &a.CreatedAt)
		if err != nil {
			return nil, err
		}

		a.Config = json.RawMessage(config)
		a.Enabled = enabled != 0
		actions = append(actions, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return actions, nil
}

// Update updates an existing action in the database.
func (r *ActionRepository) Update(a *Action) error {
	config := a.Config
	if config == nil {
		config = json.RawMessage("{}")
	}

	enabled := 0
	if a.Enabled {
		enabled = 1
	}

	result, err := r.db.Exec(
		`UPDATE actions SET gesture_id = ?, plugin_name = ?, action_name = ?, config = ?, enabled = ?
		 WHERE id = ?`,
		a.GestureID, a.PluginName, a.ActionName, string(config), enabled, a.ID,
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

// Delete removes an action from the database by its ID.
func (r *ActionRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM actions WHERE id = ?`, id)
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
