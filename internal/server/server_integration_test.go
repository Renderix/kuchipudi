package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ayusman/kuchipudi/internal/store"
)

func TestAPI_GestureWorkflow(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	s, _ := store.New(filepath.Join(tmpDir, "test.db"))
	defer s.Close()

	srv := New(Config{Store: s})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := ts.Client()

	// 1. Create a gesture
	createBody := `{"name": "test-gesture", "type": "static"}`
	resp, err := client.Post(ts.URL+"/api/gestures", "application/json", bytes.NewBufferString(createBody))
	if err != nil {
		t.Fatalf("POST /api/gestures error = %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	if created.Name != "test-gesture" {
		t.Errorf("created name = %s, want test-gesture", created.Name)
	}

	// 2. List gestures
	resp, _ = client.Get(ts.URL + "/api/gestures")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/gestures status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var listed struct {
		Gestures []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"gestures"`
	}
	json.NewDecoder(resp.Body).Decode(&listed)
	resp.Body.Close()

	if len(listed.Gestures) != 1 {
		t.Fatalf("len(gestures) = %d, want 1", len(listed.Gestures))
	}

	// 3. Get single gesture
	resp, _ = client.Get(ts.URL + "/api/gestures/" + created.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/gestures/%s status = %d, want %d", created.ID, resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	// 4. Delete gesture
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/gestures/"+created.ID, nil)
	resp, _ = client.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	resp.Body.Close()

	// 5. Verify deleted
	resp, _ = client.Get(ts.URL + "/api/gestures/" + created.ID)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET after delete status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	resp.Body.Close()
}

func TestAPI_HealthCheck(t *testing.T) {
	srv := New(Config{})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var health struct {
		Status string `json:"status"`
		Uptime string `json:"uptime"`
	}
	json.NewDecoder(resp.Body).Decode(&health)

	if health.Status != "ok" {
		t.Errorf("status = %s, want ok", health.Status)
	}
}
