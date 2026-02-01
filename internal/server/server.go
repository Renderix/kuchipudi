// Package server provides the HTTP server for the Kuchipudi gesture recognition system.
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ayusman/kuchipudi/internal/capture"
	"github.com/ayusman/kuchipudi/internal/detector"
	"github.com/ayusman/kuchipudi/internal/server/api"
	"github.com/ayusman/kuchipudi/internal/store"
)

// Config holds the server configuration.
type Config struct {
	StaticDir string
	Store     *store.Store
	Camera    *capture.Camera
	Detector  detector.Detector
}

// Server represents the HTTP server for the Kuchipudi application.
type Server struct {
	config Config
	mux    *http.ServeMux
	start  time.Time
}

// New creates a new Server with the given configuration.
func New(config Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
		start:  time.Now(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes for the server.
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)

	// Register gesture API handler if Store is configured
	if s.config.Store != nil {
		gestureHandler := api.NewGestureHandler(s.config.Store)
		samplesHandler := api.NewSamplesHandler(s.config.Store)

		// Use a wrapper to route between gestures and samples handlers
		gestureRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is a samples request: /api/gestures/{id}/samples
			if strings.HasSuffix(r.URL.Path, "/samples") {
				samplesHandler.ServeHTTP(w, r)
				return
			}
			gestureHandler.ServeHTTP(w, r)
		})

		s.mux.Handle("/api/gestures", gestureRouter)
		s.mux.Handle("/api/gestures/", gestureRouter)
	}

	// Register camera stream endpoint if Camera is configured
	if s.config.Camera != nil {
		streamHandler := NewStreamHandler(s.config.Camera)
		s.mux.Handle("/api/stream", streamHandler)
	}

	// Register landmarks WebSocket endpoint if Camera and Detector are configured
	if s.config.Camera != nil && s.config.Detector != nil {
		landmarksHandler := NewLandmarksHandler(s.config.Detector, s.config.Camera)
		s.mux.Handle("/api/landmarks", landmarksHandler)
	}

	// Serve static files if StaticDir is configured
	if s.config.StaticDir != "" {
		fs := http.FileServer(http.Dir(s.config.StaticDir))
		s.mux.Handle("/", fs)
	}
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleHealth handles GET requests to /api/health.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(s.start)

	response := map[string]interface{}{
		"status": "ok",
		"uptime": uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ListenAndServe starts the HTTP server on the given address.
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}
