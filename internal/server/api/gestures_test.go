package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ayusman/kuchipudi/internal/store"
)

// newTestStore creates a new Store with a temporary database for testing.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kuchipudi-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
	})

	return s
}

func TestGestureHandler_List(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// Create a gesture in the store
	gesture := &store.Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      store.GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}
	if err := s.Gestures().Create(gesture); err != nil {
		t.Fatalf("failed to create gesture: %v", err)
	}

	// Make a GET request to list gestures
	req := httptest.NewRequest(http.MethodGet, "/api/gestures", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var response listGesturesResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Gestures) != 1 {
		t.Errorf("expected 1 gesture, got %d", len(response.Gestures))
	}

	if response.Gestures[0].ID != "test-gesture-1" {
		t.Errorf("expected gesture ID 'test-gesture-1', got %q", response.Gestures[0].ID)
	}

	if response.Gestures[0].Name != "thumbs_up" {
		t.Errorf("expected gesture name 'thumbs_up', got %q", response.Gestures[0].Name)
	}
}

func TestGestureHandler_Create(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// Create request body
	reqBody := createGestureRequest{
		Name:      "wave",
		Type:      "dynamic",
		Tolerance: 0.20,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Make a POST request to create gesture
	req := httptest.NewRequest(http.MethodPost, "/api/gestures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var response gestureResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID == "" {
		t.Error("expected non-empty ID in response")
	}

	if response.Name != "wave" {
		t.Errorf("expected name 'wave', got %q", response.Name)
	}

	if response.Type != "dynamic" {
		t.Errorf("expected type 'dynamic', got %q", response.Type)
	}

	if response.Tolerance != 0.20 {
		t.Errorf("expected tolerance 0.20, got %f", response.Tolerance)
	}

	// Verify the gesture was persisted in the store
	created, err := s.Gestures().GetByID(response.ID)
	if err != nil {
		t.Fatalf("failed to get created gesture: %v", err)
	}

	if created.Name != "wave" {
		t.Errorf("stored gesture name mismatch: got %q, want 'wave'", created.Name)
	}
}

func TestGestureHandler_Create_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// Make a POST request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/gestures", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGestureHandler_Create_MissingName(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// Create request body without name
	reqBody := createGestureRequest{
		Type:      "static",
		Tolerance: 0.15,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/gestures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGestureHandler_Get(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// Create a gesture in the store
	gesture := &store.Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      store.GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}
	if err := s.Gestures().Create(gesture); err != nil {
		t.Fatalf("failed to create gesture: %v", err)
	}

	// Make a GET request to get the gesture
	req := httptest.NewRequest(http.MethodGet, "/api/gestures/test-gesture-1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response gestureResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != "test-gesture-1" {
		t.Errorf("expected ID 'test-gesture-1', got %q", response.ID)
	}

	if response.Name != "thumbs_up" {
		t.Errorf("expected name 'thumbs_up', got %q", response.Name)
	}
}

func TestGestureHandler_Get_NotFound(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/api/gestures/non-existent", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGestureHandler_Update(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// Create a gesture in the store
	gesture := &store.Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      store.GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}
	if err := s.Gestures().Create(gesture); err != nil {
		t.Fatalf("failed to create gesture: %v", err)
	}

	// Make a PUT request to update the gesture
	updateReq := updateGestureRequest{
		Name:      "thumbs_up_v2",
		Type:      "dynamic",
		Tolerance: 0.25,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/gestures/test-gesture-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response gestureResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "thumbs_up_v2" {
		t.Errorf("expected name 'thumbs_up_v2', got %q", response.Name)
	}

	if response.Type != "dynamic" {
		t.Errorf("expected type 'dynamic', got %q", response.Type)
	}

	// Verify the update was persisted
	updated, _ := s.Gestures().GetByID("test-gesture-1")
	if updated.Name != "thumbs_up_v2" {
		t.Errorf("stored gesture name not updated: got %q", updated.Name)
	}
}

func TestGestureHandler_Update_NotFound(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	updateReq := updateGestureRequest{
		Name:      "updated",
		Type:      "static",
		Tolerance: 0.15,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/gestures/non-existent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGestureHandler_Delete(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// Create a gesture in the store
	gesture := &store.Gesture{
		ID:        "test-gesture-1",
		Name:      "thumbs_up",
		Type:      store.GestureTypeStatic,
		Tolerance: 0.15,
		Samples:   10,
	}
	if err := s.Gestures().Create(gesture); err != nil {
		t.Fatalf("failed to create gesture: %v", err)
	}

	// Make a DELETE request
	req := httptest.NewRequest(http.MethodDelete, "/api/gestures/test-gesture-1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify 204 No Content
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	// Verify the gesture is deleted - GET should return 404
	req = httptest.NewRequest(http.MethodGet, "/api/gestures/test-gesture-1", nil)
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d after delete, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGestureHandler_Delete_NotFound(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	req := httptest.NewRequest(http.MethodDelete, "/api/gestures/non-existent", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGestureHandler_MethodNotAllowed(t *testing.T) {
	s := newTestStore(t)
	handler := NewGestureHandler(s)

	// PATCH is not allowed on the collection endpoint
	req := httptest.NewRequest(http.MethodPatch, "/api/gestures", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
