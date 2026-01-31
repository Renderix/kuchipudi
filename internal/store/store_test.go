package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore_CreatesDatabase(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "kuchipudi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Verify the database file doesn't exist yet
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatal("database file should not exist before creating store")
	}

	// Create the store
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Verify the database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file should exist after creating store")
	}
}

func TestNewStore_RunsMigrations(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "kuchipudi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create the store
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Verify that the gestures table exists by querying it
	tables := []string{"gestures", "gesture_landmarks", "gesture_paths", "actions", "settings"}
	for _, table := range tables {
		var name string
		err := s.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q should exist after migrations: %v", table, err)
		}
	}
}

func TestStore_Close(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "kuchipudi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create the store
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Close should not return an error
	if err := s.Close(); err != nil {
		t.Errorf("close should not return error: %v", err)
	}

	// After closing, DB operations should fail
	_, err = s.DB().Exec("SELECT 1")
	if err == nil {
		t.Error("DB operations should fail after close")
	}
}

func TestStore_ForeignKeysEnabled(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "kuchipudi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create the store
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Check that foreign keys are enabled
	var fkEnabled int
	err = s.DB().QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("failed to check foreign keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("foreign keys should be enabled")
	}
}

func TestStore_IndexesCreated(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "kuchipudi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create the store
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Verify indexes exist
	indexes := []string{
		"idx_gesture_landmarks_gesture_id",
		"idx_gesture_paths_gesture_id",
		"idx_actions_gesture_id",
	}
	for _, idx := range indexes {
		var name string
		err := s.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
			idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("index %q should exist after migrations: %v", idx, err)
		}
	}
}
