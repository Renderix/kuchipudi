// Package server provides the HTTP server for the Kuchipudi gesture recognition system.
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ayusman/kuchipudi/internal/server/api"
	"github.com/ayusman/kuchipudi/internal/store"
)

// Config holds the server configuration.
type Config struct {
	StaticDir string
	Store     *store.Store
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
		s.mux.Handle("/api/gestures", gestureHandler)
		s.mux.Handle("/api/gestures/", gestureHandler)
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
