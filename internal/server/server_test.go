package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServer_Health(t *testing.T) {
	s := New(Config{})

	t.Run("returns 200 with JSON response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()

		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["status"] != "ok" {
			t.Errorf("expected status 'ok', got %v", response["status"])
		}

		if _, exists := response["uptime"]; !exists {
			t.Error("expected 'uptime' field in response")
		}
	})

	t.Run("only allows GET method", func(t *testing.T) {
		methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/api/health", nil)
			rec := httptest.NewRecorder()

			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("method %s: expected status %d, got %d", method, http.StatusMethodNotAllowed, rec.Code)
			}
		}
	})
}

func TestServer_NotFound(t *testing.T) {
	s := New(Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestServer_StaticFiles(t *testing.T) {
	// Create a temporary directory with a static file
	tmpDir, err := os.MkdirTemp("", "kuchipudi-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test HTML file
	testContent := "<html><body>Hello, World!</body></html>"
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a CSS file for testing direct file access
	cssContent := "body { color: red; }"
	if err := os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte(cssContent), 0644); err != nil {
		t.Fatalf("failed to create test CSS file: %v", err)
	}

	s := New(Config{StaticDir: tmpDir})

	t.Run("serves index.html at root path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		if rec.Body.String() != testContent {
			t.Errorf("expected body %q, got %q", testContent, rec.Body.String())
		}
	})

	t.Run("serves static files from configured directory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
		rec := httptest.NewRecorder()

		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		if rec.Body.String() != cssContent {
			t.Errorf("expected body %q, got %q", cssContent, rec.Body.String())
		}
	})

	t.Run("returns 404 for non-existent static files", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent.html", nil)
		rec := httptest.NewRecorder()

		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestServer_NoStaticDir(t *testing.T) {
	s := New(Config{})

	t.Run("root path returns 404 when no static dir configured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("creates server with config", func(t *testing.T) {
		cfg := Config{StaticDir: "/some/path"}
		s := New(cfg)

		if s == nil {
			t.Fatal("expected non-nil server")
		}

		if s.config.StaticDir != cfg.StaticDir {
			t.Errorf("expected StaticDir %s, got %s", cfg.StaticDir, s.config.StaticDir)
		}
	})

	t.Run("server implements http.Handler", func(t *testing.T) {
		s := New(Config{})
		var _ http.Handler = s
	})
}
