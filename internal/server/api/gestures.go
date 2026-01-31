// Package api provides HTTP API handlers for the Kuchipudi gesture recognition system.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ayusman/kuchipudi/internal/store"
)

// GestureHandler handles HTTP requests for gesture resources.
type GestureHandler struct {
	store *store.Store
}

// NewGestureHandler creates a new GestureHandler with the given store.
func NewGestureHandler(s *store.Store) *GestureHandler {
	return &GestureHandler{store: s}
}

// ServeHTTP implements the http.Handler interface and routes requests to appropriate methods.
func (h *GestureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse the path to determine if this is a collection or item request
	// Expected paths: /api/gestures or /api/gestures/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/gestures")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		// Collection endpoint: /api/gestures
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Item endpoint: /api/gestures/{id}
	id := path
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Request and response types

type createGestureRequest struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Tolerance float64 `json:"tolerance"`
}

type updateGestureRequest struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Tolerance float64 `json:"tolerance"`
}

type gestureResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Tolerance float64 `json:"tolerance"`
	Samples   int     `json:"samples"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type listGesturesResponse struct {
	Gestures []gestureResponse `json:"gestures"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// toResponse converts a store.Gesture to a gestureResponse.
func toResponse(g *store.Gesture) gestureResponse {
	return gestureResponse{
		ID:        g.ID,
		Name:      g.Name,
		Type:      string(g.Type),
		Tolerance: g.Tolerance,
		Samples:   g.Samples,
		CreatedAt: g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// list handles GET /api/gestures and returns all gestures.
func (h *GestureHandler) list(w http.ResponseWriter, r *http.Request) {
	gestures, err := h.store.Gestures().List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list gestures")
		return
	}

	response := listGesturesResponse{
		Gestures: make([]gestureResponse, 0, len(gestures)),
	}

	for _, g := range gestures {
		response.Gestures = append(response.Gestures, toResponse(g))
	}

	writeJSON(w, http.StatusOK, response)
}

// get handles GET /api/gestures/{id} and returns a single gesture.
func (h *GestureHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	gesture, err := h.store.Gestures().GetByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Gesture not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get gesture")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(gesture))
}

// create handles POST /api/gestures and creates a new gesture.
func (h *GestureHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createGestureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Set default type if not provided
	gestureType := store.GestureType(req.Type)
	if gestureType == "" {
		gestureType = store.GestureTypeStatic
	}

	// Validate gesture type
	if gestureType != store.GestureTypeStatic && gestureType != store.GestureTypeDynamic {
		writeError(w, http.StatusBadRequest, "Invalid gesture type")
		return
	}

	// Set default tolerance if not provided
	tolerance := req.Tolerance
	if tolerance == 0 {
		tolerance = 0.15
	}

	gesture := &store.Gesture{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      gestureType,
		Tolerance: tolerance,
		Samples:   0,
	}

	if err := h.store.Gestures().Create(gesture); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create gesture")
		return
	}

	writeJSON(w, http.StatusCreated, toResponse(gesture))
}

// update handles PUT /api/gestures/{id} and updates an existing gesture.
func (h *GestureHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	// First, get the existing gesture
	gesture, err := h.store.Gestures().GetByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Gesture not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get gesture")
		return
	}

	var req updateGestureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Update fields if provided
	if req.Name != "" {
		gesture.Name = req.Name
	}
	if req.Type != "" {
		gestureType := store.GestureType(req.Type)
		if gestureType != store.GestureTypeStatic && gestureType != store.GestureTypeDynamic {
			writeError(w, http.StatusBadRequest, "Invalid gesture type")
			return
		}
		gesture.Type = gestureType
	}
	if req.Tolerance != 0 {
		gesture.Tolerance = req.Tolerance
	}

	if err := h.store.Gestures().Update(gesture); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update gesture")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(gesture))
}

// delete handles DELETE /api/gestures/{id} and removes a gesture.
func (h *GestureHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	err := h.store.Gestures().Delete(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Gesture not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete gesture")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
