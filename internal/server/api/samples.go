package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ayusman/kuchipudi/internal/store"
)

// SamplesHandler handles HTTP requests for gesture sample resources.
type SamplesHandler struct {
	store *store.Store
}

// NewSamplesHandler creates a new SamplesHandler with the given store.
func NewSamplesHandler(s *store.Store) *SamplesHandler {
	return &SamplesHandler{store: s}
}

// ServeHTTP implements the http.Handler interface.
// Expected paths: /api/gestures/{id}/samples
func (h *SamplesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse gesture ID from path: /api/gestures/{id}/samples
	path := strings.TrimPrefix(r.URL.Path, "/api/gestures/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 || parts[1] != "samples" {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}

	gestureID := parts[0]

	switch r.Method {
	case http.MethodGet:
		h.list(w, r, gestureID)
	case http.MethodPost:
		h.create(w, r, gestureID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Request types

type createSamplesRequest struct {
	Samples []json.RawMessage `json:"samples"`
}

// Response types

type sampleResponse struct {
	ID          int64           `json:"id"`
	GestureID   string          `json:"gesture_id"`
	SampleIndex int             `json:"sample_index"`
	Data        json.RawMessage `json:"data"`
	CreatedAt   string          `json:"created_at"`
}

type listSamplesResponse struct {
	Samples []sampleResponse `json:"samples"`
}

// list handles GET /api/gestures/{id}/samples
func (h *SamplesHandler) list(w http.ResponseWriter, r *http.Request, gestureID string) {
	samples, err := h.store.Samples().GetByGestureID(gestureID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list samples")
		return
	}

	response := listSamplesResponse{
		Samples: make([]sampleResponse, 0, len(samples)),
	}

	for _, s := range samples {
		response.Samples = append(response.Samples, sampleResponse{
			ID:          s.ID,
			GestureID:   s.GestureID,
			SampleIndex: s.SampleIndex,
			Data:        s.Data,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// create handles POST /api/gestures/{id}/samples
func (h *SamplesHandler) create(w http.ResponseWriter, r *http.Request, gestureID string) {
	// Verify gesture exists
	_, err := h.store.Gestures().GetByID(gestureID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "Gesture not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to verify gesture")
		return
	}

	var req createSamplesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(req.Samples) == 0 {
		writeError(w, http.StatusBadRequest, "At least one sample is required")
		return
	}

	if err := h.store.Samples().Create(gestureID, req.Samples); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save samples")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
