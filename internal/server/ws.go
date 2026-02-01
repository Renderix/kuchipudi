// Package server provides the HTTP server for the Kuchipudi gesture recognition system.
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ayusman/kuchipudi/internal/capture"
	"github.com/ayusman/kuchipudi/internal/detector"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow local connections
	},
}

// LandmarksHandler broadcasts real-time hand landmarks via WebSocket.
type LandmarksHandler struct {
	detector detector.Detector
	camera   *capture.Camera
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex
}

// NewLandmarksHandler creates a new LandmarksHandler with the given detector and camera.
func NewLandmarksHandler(d detector.Detector, c *capture.Camera) *LandmarksHandler {
	h := &LandmarksHandler{
		detector: d,
		camera:   c,
		clients:  make(map[*websocket.Conn]bool),
	}
	go h.broadcast()
	return h
}

// ServeHTTP handles WebSocket upgrade requests.
func (h *LandmarksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
	}()

	// Keep connection alive by reading messages
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// broadcast sends landmark data to all connected clients.
func (h *LandmarksHandler) broadcast() {
	ticker := time.NewTicker(66 * time.Millisecond) // ~15 FPS
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		if len(h.clients) == 0 {
			h.mu.RUnlock()
			continue
		}
		h.mu.RUnlock()

		frame, err := h.camera.ReadFrame()
		if err != nil {
			continue
		}

		hands, err := h.detector.Detect(frame)
		frame.Close()
		if err != nil {
			continue
		}

		msg, _ := json.Marshal(map[string]any{
			"hands":     hands,
			"timestamp": time.Now().UnixMilli(),
		})

		h.mu.RLock()
		for conn := range h.clients {
			conn.WriteMessage(websocket.TextMessage, msg)
		}
		h.mu.RUnlock()
	}
}
