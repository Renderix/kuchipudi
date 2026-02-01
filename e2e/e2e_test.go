package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayusman/kuchipudi/internal/app"
	"github.com/ayusman/kuchipudi/internal/detector"
	"github.com/ayusman/kuchipudi/internal/gesture"
	"github.com/ayusman/kuchipudi/internal/server"
	"github.com/ayusman/kuchipudi/internal/store"
)

func TestE2E_CompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "data.db")

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	defer s.Close()

	srv := server.New(server.Config{Store: s})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := ts.Client()

	t.Run("CreateGesture", func(t *testing.T) {
		resp, err := client.Post(
			ts.URL+"/api/gestures",
			"application/json",
			strings.NewReader(`{"name": "wave", "type": "dynamic"}`),
		)
		if err != nil {
			t.Fatalf("create gesture error = %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
		}
	})

	application := app.New(app.Config{
		Store:        s,
		PluginDir:    filepath.Join(tmpDir, "plugins"),
		MotionThresh: 0.05,
	})

	mockDetector := detector.NewMockDetector()
	application.SetDetector(mockDetector)

	t.Run("LoadGestures", func(t *testing.T) {
		if err := application.LoadGestures(); err != nil {
			t.Fatalf("LoadGestures() error = %v", err)
		}
	})

	t.Run("DetectGesture", func(t *testing.T) {
		mockDetector.SetHands([]detector.HandLandmarks{detector.ThumbsUpLandmarks()})

		thumbsUp := detector.ThumbsUpLandmarks()
		normalized := thumbsUp.Normalize()
		application.StaticMatcher().AddTemplate(&gesture.Template{
			ID:        "test-thumbs-up",
			Name:      "Test Thumbs Up",
			Type:      gesture.TypeStatic,
			Landmarks: normalized.Points[:],
			Tolerance: 0.3,
		})

		hands, _ := mockDetector.Detect(nil)
		if len(hands) == 0 {
			t.Fatal("no hands detected")
		}

		matches := application.StaticMatcher().Match(&hands[0])
		if len(matches) == 0 {
			t.Error("expected gesture to match")
		}
	})

	t.Run("APIStillWorks", func(t *testing.T) {
		resp, _ := client.Get(ts.URL + "/api/health")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("health check failed after app operations")
		}
		resp.Body.Close()
	})
}

func TestE2E_GestureRecordAndMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	tmpDir := t.TempDir()
	s, _ := store.New(filepath.Join(tmpDir, "data.db"))
	defer s.Close()

	g := &store.Gesture{
		ID:        "recorded-1",
		Name:      "Custom Gesture",
		Type:      store.GestureTypeStatic,
		Tolerance: 0.25,
	}
	s.Gestures().Create(g)

	landmarks := detector.ThumbsUpLandmarks()
	normalized := landmarks.Normalize()
	template := &gesture.Template{
		ID:        g.ID,
		Name:      g.Name,
		Type:      gesture.TypeStatic,
		Landmarks: normalized.Points[:],
		Tolerance: g.Tolerance,
	}

	matcher := gesture.NewStaticMatcher()
	matcher.AddTemplate(template)

	input := detector.ThumbsUpLandmarks()
	matches := matcher.Match(&input)

	if len(matches) == 0 {
		t.Error("recorded gesture should match input")
	}

	if matches[0].Score < 0.9 {
		t.Errorf("score = %f, expected > 0.9 for identical gesture", matches[0].Score)
	}
}

func TestE2E_ActionBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	tmpDir := t.TempDir()
	s, _ := store.New(filepath.Join(tmpDir, "data.db"))
	defer s.Close()

	srv := server.New(server.Config{Store: s})
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := ts.Client()

	resp, err := client.Post(
		ts.URL+"/api/gestures",
		"application/json",
		strings.NewReader(`{"name": "test-gesture", "type": "static"}`),
	)
	if err != nil {
		t.Fatalf("create gesture error = %v", err)
	}

	var gestureResp struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&gestureResp)
	resp.Body.Close()

	actionReq := map[string]interface{}{
		"gesture_id":  gestureResp.ID,
		"plugin_name": "system-control",
		"action_name": "volume_up",
		"enabled":     true,
	}
	actionBody, _ := json.Marshal(actionReq)

	resp, err = client.Post(
		ts.URL+"/api/actions",
		"application/json",
		strings.NewReader(string(actionBody)),
	)
	if err != nil {
		t.Fatalf("create action error = %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("create action status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	resp.Body.Close()

	resp, err = client.Get(ts.URL + "/api/actions")
	if err != nil {
		t.Fatalf("list actions error = %v", err)
	}

	var listResp struct {
		Actions []struct {
			ID         string `json:"id"`
			GestureID  string `json:"gesture_id"`
			PluginName string `json:"plugin_name"`
			ActionName string `json:"action_name"`
			Enabled    bool   `json:"enabled"`
		} `json:"actions"`
	}
	json.NewDecoder(resp.Body).Decode(&listResp)
	resp.Body.Close()

	if len(listResp.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(listResp.Actions))
	}

	if listResp.Actions[0].GestureID != gestureResp.ID {
		t.Errorf("action gesture_id mismatch: got %s, want %s", listResp.Actions[0].GestureID, gestureResp.ID)
	}
}
