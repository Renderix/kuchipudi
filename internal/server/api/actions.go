package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ayusman/kuchipudi/internal/store"
)

// ActionHandler handles HTTP requests for action resources.
type ActionHandler struct {
	store *store.Store
}

// NewActionHandler creates a new ActionHandler with the given store.
func NewActionHandler(s *store.Store) *ActionHandler {
	return &ActionHandler{store: s}
}

// ServeHTTP implements the http.Handler interface and routes requests to appropriate methods.
func (h *ActionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse the path to determine if this is a collection or item request
	// Expected paths: /api/actions or /api/actions/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/actions")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		// Collection endpoint: /api/actions
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

	// Item endpoint: /api/actions/{id}
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

type createActionRequest struct {
	GestureID  string          `json:"gesture_id"`
	PluginName string          `json:"plugin_name"`
	ActionName string          `json:"action_name"`
	Config     json.RawMessage `json:"config"`
}

type updateActionRequest struct {
	GestureID  string          `json:"gesture_id"`
	PluginName string          `json:"plugin_name"`
	ActionName string          `json:"action_name"`
	Config     json.RawMessage `json:"config"`
	Enabled    *bool           `json:"enabled"`
}

type actionResponse struct {
	ID         string          `json:"id"`
	GestureID  string          `json:"gesture_id"`
	PluginName string          `json:"plugin_name"`
	ActionName string          `json:"action_name"`
	Config     json.RawMessage `json:"config"`
	Enabled    bool            `json:"enabled"`
	CreatedAt  string          `json:"created_at"`
}

type listActionsResponse struct {
	Actions []actionResponse `json:"actions"`
}

// toActionResponse converts a store.Action to an actionResponse.
func toActionResponse(a *store.Action) actionResponse {
	config := a.Config
	if config == nil {
		config = json.RawMessage("{}")
	}
	return actionResponse{
		ID:         a.ID,
		GestureID:  a.GestureID,
		PluginName: a.PluginName,
		ActionName: a.ActionName,
		Config:     config,
		Enabled:    a.Enabled,
		CreatedAt:  a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// list handles GET /api/actions and returns all actions.
func (h *ActionHandler) list(w http.ResponseWriter, r *http.Request) {
	actions, err := h.store.Actions().List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list actions")
		return
	}

	response := listActionsResponse{
		Actions: make([]actionResponse, 0, len(actions)),
	}

	for _, a := range actions {
		response.Actions = append(response.Actions, toActionResponse(a))
	}

	writeJSON(w, http.StatusOK, response)
}

// get handles GET /api/actions/{id} and returns a single action.
func (h *ActionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	action, err := h.store.Actions().GetByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Action not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get action")
		return
	}

	writeJSON(w, http.StatusOK, toActionResponse(action))
}

// create handles POST /api/actions and creates a new action.
func (h *ActionHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate required fields
	if req.GestureID == "" {
		writeError(w, http.StatusBadRequest, "gesture_id is required")
		return
	}
	if req.PluginName == "" {
		writeError(w, http.StatusBadRequest, "plugin_name is required")
		return
	}
	if req.ActionName == "" {
		writeError(w, http.StatusBadRequest, "action_name is required")
		return
	}

	// Verify gesture exists
	_, err := h.store.Gestures().GetByID(req.GestureID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "Gesture not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to verify gesture")
		return
	}

	// Check for duplicate binding
	existing, err := h.store.Actions().GetByGestureID(req.GestureID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check existing action")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "Action already bound to this gesture")
		return
	}

	config := req.Config
	if config == nil {
		config = json.RawMessage("{}")
	}

	action := &store.Action{
		ID:         uuid.New().String(),
		GestureID:  req.GestureID,
		PluginName: req.PluginName,
		ActionName: req.ActionName,
		Config:     config,
		Enabled:    true,
	}

	if err := h.store.Actions().Create(action); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create action")
		return
	}

	writeJSON(w, http.StatusCreated, toActionResponse(action))
}

// update handles PUT /api/actions/{id} and updates an existing action.
func (h *ActionHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	// First, get the existing action
	action, err := h.store.Actions().GetByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Action not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get action")
		return
	}

	var req updateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Update fields if provided
	if req.GestureID != "" {
		// Verify new gesture exists
		_, err := h.store.Gestures().GetByID(req.GestureID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusBadRequest, "Gesture not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to verify gesture")
			return
		}
		action.GestureID = req.GestureID
	}
	if req.PluginName != "" {
		action.PluginName = req.PluginName
	}
	if req.ActionName != "" {
		action.ActionName = req.ActionName
	}
	if req.Config != nil {
		action.Config = req.Config
	}
	if req.Enabled != nil {
		action.Enabled = *req.Enabled
	}

	if err := h.store.Actions().Update(action); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update action")
		return
	}

	writeJSON(w, http.StatusOK, toActionResponse(action))
}

// delete handles DELETE /api/actions/{id} and removes an action.
func (h *ActionHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	err := h.store.Actions().Delete(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Action not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete action")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
