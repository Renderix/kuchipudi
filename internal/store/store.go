// Package store provides SQLite database storage for the Kuchipudi gesture recognition system.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store represents a SQLite database connection for storing gestures and related data.
type Store struct {
	db   *sql.DB
	path string
}

// New creates a new Store with the given database path.
// It opens the database connection, enables foreign keys, and runs migrations.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	s := &Store{
		db:   db,
		path: dbPath,
	}

	// Run migrations
	if err := s.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}
