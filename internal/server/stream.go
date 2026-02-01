// Package server provides the HTTP server for the Kuchipudi gesture recognition system.
package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ayusman/kuchipudi/internal/capture"
	"gocv.io/x/gocv"
)

// StreamHandler serves MJPEG frames from the camera.
type StreamHandler struct {
	camera capture.Camera
}

// NewStreamHandler creates a new StreamHandler with the given camera.
func NewStreamHandler(camera capture.Camera) *StreamHandler {
	return &StreamHandler{camera: camera}
}

// ServeHTTP streams MJPEG frames to connected clients.
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		frame, err := h.camera.ReadFrame()
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Encode as JPEG
		buf, err := gocv.IMEncode(".jpg", *frame)
		frame.Close()
		if err != nil {
			continue
		}

		// Write MJPEG frame
		fmt.Fprintf(w, "--frame\r\n")
		fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
		fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", buf.Len())
		w.Write(buf.GetBytes())
		fmt.Fprintf(w, "\r\n")
		buf.Close()

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		time.Sleep(66 * time.Millisecond) // ~15 FPS
	}
}
