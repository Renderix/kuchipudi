package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestStore creates a new Store with an in-memory database for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kuchipudi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
	})

	return s
}

func TestGestureRepository_Create(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	gesture := &Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}

	// Create the gesture
	err := repo.Create(gesture)
	if err != nil {
		t.Fatalf("failed to create gesture: %v", err)
	}

	// Verify CreatedAt and UpdatedAt are set
	if gesture.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set after create")
	}
	if gesture.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set after create")
	}

	// Retrieve the gesture by ID
	retrieved, err := repo.GetByID("test-gesture-1")
	if err != nil {
		t.Fatalf("failed to get gesture by ID: %v", err)
	}

	// Verify all fields match
	if retrieved.ID != gesture.ID {
		t.Errorf("ID mismatch: got %q, want %q", retrieved.ID, gesture.ID)
	}
	if retrieved.Name != gesture.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, gesture.Name)
	}
	if retrieved.Type != gesture.Type {
		t.Errorf("Type mismatch: got %q, want %q", retrieved.Type, gesture.Type)
	}
	if retrieved.Tolerance != gesture.Tolerance {
		t.Errorf("Tolerance mismatch: got %f, want %f", retrieved.Tolerance, gesture.Tolerance)
	}
	if retrieved.Samples != gesture.Samples {
		t.Errorf("Samples mismatch: got %d, want %d", retrieved.Samples, gesture.Samples)
	}

	// Retrieve the gesture by name
	retrievedByName, err := repo.GetByName("thumbs_up")
	if err != nil {
		t.Fatalf("failed to get gesture by name: %v", err)
	}
	if retrievedByName.ID != gesture.ID {
		t.Errorf("GetByName returned wrong gesture: got ID %q, want %q", retrievedByName.ID, gesture.ID)
	}
}

func TestGestureRepository_Create_DuplicateName(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	gesture1 := &Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}

	gesture2 := &Gesture{
		ID:        "test-gesture-2",
		Name:      "thumbs_up", // Same name
		Type:      GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   5,
	}

	// Create the first gesture
	if err := repo.Create(gesture1); err != nil {
		t.Fatalf("failed to create first gesture: %v", err)
	}

	// Creating second gesture with same name should fail
	err := repo.Create(gesture2)
	if err == nil {
		t.Error("creating gesture with duplicate name should fail")
	}
}

func TestGestureRepository_List(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	// Create multiple gestures
	gestures := []*Gesture{
		{ID: "gesture-1", Name: "thumbs_up", Type: GestureTypeStatic, Tolerance: 0.15, Samples: 10},
		{ID: "gesture-2", Name: "wave", Type: GestureTypeDynamic, Tolerance: 0.20, Samples: 5},
		{ID: "gesture-3", Name: "peace", Type: GestureTypeStatic, Tolerance: 0.10, Samples: 15},
	}

	for _, g := range gestures {
		if err := repo.Create(g); err != nil {
			t.Fatalf("failed to create gesture %q: %v", g.Name, err)
		}
	}

	// List all gestures
	list, err := repo.List()
	if err != nil {
		t.Fatalf("failed to list gestures: %v", err)
	}

	if len(list) != len(gestures) {
		t.Errorf("expected %d gestures, got %d", len(gestures), len(list))
	}

	// Verify all gestures are present
	nameMap := make(map[string]bool)
	for _, g := range list {
		nameMap[g.Name] = true
	}
	for _, g := range gestures {
		if !nameMap[g.Name] {
			t.Errorf("gesture %q not found in list", g.Name)
		}
	}
}

func TestGestureRepository_Delete(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	gesture := &Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}

	// Create the gesture
	if err := repo.Create(gesture); err != nil {
		t.Fatalf("failed to create gesture: %v", err)
	}

	// Verify it exists
	_, err := repo.GetByID("test-gesture-1")
	if err != nil {
		t.Fatalf("gesture should exist after create: %v", err)
	}

	// Delete the gesture
	err = repo.Delete("test-gesture-1")
	if err != nil {
		t.Fatalf("failed to delete gesture: %v", err)
	}

	// Verify it's gone
	_, err = repo.GetByID("test-gesture-1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestGestureRepository_Delete_NotFound(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	// Delete a non-existent gesture should return ErrNotFound
	err := repo.Delete("non-existent-id")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for non-existent gesture, got: %v", err)
	}
}

func TestGestureRepository_GetByID_NotFound(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	_, err := repo.GetByID("non-existent-id")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGestureRepository_GetByName_NotFound(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	_, err := repo.GetByName("non-existent-name")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGestureRepository_Update(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	gesture := &Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}

	// Create the gesture
	if err := repo.Create(gesture); err != nil {
		t.Fatalf("failed to create gesture: %v", err)
	}

	originalUpdatedAt := gesture.UpdatedAt

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(10 * time.Millisecond)

	// Update the gesture
	gesture.Name = "thumbs_up_v2"
	gesture.Tolerance = 0.20
	gesture.Samples = 20

	if err := repo.Update(gesture); err != nil {
		t.Fatalf("failed to update gesture: %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.GetByID("test-gesture-1")
	if err != nil {
		t.Fatalf("failed to get gesture after update: %v", err)
	}

	if retrieved.Name != "thumbs_up_v2" {
		t.Errorf("Name not updated: got %q, want %q", retrieved.Name, "thumbs_up_v2")
	}
	if retrieved.Tolerance != 0.20 {
		t.Errorf("Tolerance not updated: got %f, want %f", retrieved.Tolerance, 0.20)
	}
	if retrieved.Samples != 20 {
		t.Errorf("Samples not updated: got %d, want %d", retrieved.Samples, 20)
	}
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated after Update")
	}
}

func TestGestureRepository_Update_NotFound(t *testing.T) {
	s := newTestStore(t)
	repo := s.Gestures()

	gesture := &Gesture{
		ID:        "non-existent-id",
		Name:      "test",
		Type:      GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}

	err := repo.Update(gesture)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for non-existent gesture, got: %v", err)
	}
}

func TestGestureType_Constants(t *testing.T) {
	// Verify the gesture type constants
	if GestureTypeStatic != "static" {
		t.Errorf("GestureTypeStatic should be 'static', got %q", GestureTypeStatic)
	}
	if GestureTypeDynamic != "dynamic" {
		t.Errorf("GestureTypeDynamic should be 'dynamic', got %q", GestureTypeDynamic)
	}
}
