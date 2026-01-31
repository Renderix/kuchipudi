# Gesture Recording UI Plan

> **For Claude:** Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a web-based gesture recording interface that lets users train custom gestures by recording 3-5 samples via webcam.

**Architecture:** WebRTC for camera access in browser, MJPEG stream for landmark overlay, REST API for saving gesture samples. Backend stores normalized landmarks/paths in SQLite.

**Tech Stack:** WebRTC (browser camera), Canvas API (landmark visualization), existing REST API, SQLite

---

## Phase 1: Camera Streaming

### Task 1: Add MJPEG Stream Endpoint

**Files:**
- Create: `internal/server/stream.go`
- Create: `internal/server/stream_test.go`
- Modify: `internal/server/server.go`

**Step 1: Write stream handler**

```go
// internal/server/stream.go
package server

import (
    "fmt"
    "net/http"
    "time"

    "gocv.io/x/gocv"
)

type StreamHandler struct {
    camera *capture.Camera
}

func NewStreamHandler(camera *capture.Camera) *StreamHandler {
    return &StreamHandler{camera: camera}
}

// ServeHTTP streams MJPEG frames
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
```

**Step 2: Register in server.go**

```go
func (s *Server) setupRoutes() {
    // ... existing routes

    // Camera stream (requires camera to be set)
    if s.config.Camera != nil {
        streamHandler := NewStreamHandler(s.config.Camera)
        s.mux.Handle("/api/stream", streamHandler)
    }
}
```

**Step 3: Commit**

```bash
git add internal/server/stream.go internal/server/server.go
git commit -m "feat: add MJPEG camera stream endpoint"
```

---

### Task 2: Add Landmarks WebSocket

**Files:**
- Create: `internal/server/ws.go`
- Modify: `internal/server/server.go`

**Step 1: Add gorilla/websocket dependency**

```bash
go get github.com/gorilla/websocket
```

**Step 2: Write WebSocket handler**

```go
// internal/server/ws.go
package server

import (
    "encoding/json"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "github.com/ayusman/kuchipudi/internal/detector"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow local connections
    },
}

type LandmarksHandler struct {
    detector detector.Detector
    camera   *capture.Camera
    clients  map[*websocket.Conn]bool
    mu       sync.RWMutex
}

func NewLandmarksHandler(d detector.Detector, c *capture.Camera) *LandmarksHandler {
    h := &LandmarksHandler{
        detector: d,
        camera:   c,
        clients:  make(map[*websocket.Conn]bool),
    }
    go h.broadcast()
    return h
}

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

    // Keep connection alive
    for {
        if _, _, err := conn.ReadMessage(); err != nil {
            break
        }
    }
}

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
```

**Step 3: Register in server.go**

```go
s.mux.Handle("/api/landmarks", landmarksHandler)
```

**Step 4: Commit**

```bash
git add internal/server/ws.go internal/server/server.go go.mod go.sum
git commit -m "feat: add WebSocket endpoint for real-time landmarks"
```

---

## Phase 2: Recording UI

### Task 3: Create Recording Page

**Files:**
- Create: `web/record.html`
- Modify: `web/js/app.js`

**Step 1: Write record.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Record Gesture - Kuchipudi</title>
    <link rel="stylesheet" href="/css/style.css">
    <style>
        .record-container {
            display: flex;
            gap: 2rem;
            margin-top: 2rem;
        }
        .video-panel {
            flex: 2;
            position: relative;
        }
        .video-panel video,
        .video-panel canvas {
            width: 100%;
            border-radius: 8px;
        }
        .video-panel canvas {
            position: absolute;
            top: 0;
            left: 0;
            pointer-events: none;
        }
        .control-panel {
            flex: 1;
            background: var(--bg-light);
            padding: 1.5rem;
            border-radius: 8px;
        }
        .sample-list {
            margin: 1rem 0;
        }
        .sample-item {
            display: flex;
            align-items: center;
            padding: 0.5rem;
            background: var(--bg);
            margin-bottom: 0.5rem;
            border-radius: 4px;
        }
        .sample-item.recording {
            border: 2px solid var(--primary);
            animation: pulse 1s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.7; }
        }
        .record-btn {
            width: 100%;
            padding: 1rem;
            font-size: 1.2rem;
            margin-top: 1rem;
        }
        .record-btn.recording {
            background: #e74c3c;
        }
        .instructions {
            color: var(--text-muted);
            margin-bottom: 1rem;
        }
    </style>
</head>
<body>
    <header>
        <h1>👋 Record Gesture</h1>
        <nav>
            <a href="/">← Back to Dashboard</a>
        </nav>
    </header>

    <main>
        <div class="record-container">
            <div class="video-panel">
                <video id="camera" autoplay muted playsinline></video>
                <canvas id="overlay"></canvas>
            </div>

            <div class="control-panel">
                <h2>New Gesture</h2>

                <label>
                    Gesture Name
                    <input type="text" id="gesture-name" placeholder="e.g., thumbs-up">
                </label>

                <label>
                    Type
                    <select id="gesture-type">
                        <option value="static">Static (pose)</option>
                        <option value="dynamic">Dynamic (movement)</option>
                    </select>
                </label>

                <div class="instructions">
                    <p id="static-instructions">Hold your hand pose steady and click Record to capture.</p>
                    <p id="dynamic-instructions" style="display:none">Perform the gesture motion. Recording will capture 2 seconds of movement.</p>
                </div>

                <h3>Samples (0/5)</h3>
                <div class="sample-list" id="sample-list">
                    <p class="text-muted">No samples recorded yet</p>
                </div>

                <button class="primary record-btn" id="record-btn" disabled>
                    Record Sample
                </button>

                <button class="primary" id="save-btn" disabled style="margin-top: 0.5rem; width: 100%;">
                    Save Gesture
                </button>
            </div>
        </div>
    </main>

    <script src="/js/record.js"></script>
</body>
</html>
```

**Step 2: Commit**

```bash
git add web/record.html
git commit -m "feat: add gesture recording page HTML"
```

---

### Task 4: Create Recording JavaScript

**Files:**
- Create: `web/js/record.js`

**Step 1: Write record.js**

```javascript
// web/js/record.js

const API_BASE = '/api';
let ws = null;
let recording = false;
let samples = [];
let currentLandmarks = null;
let pathBuffer = [];

// DOM elements
const video = document.getElementById('camera');
const canvas = document.getElementById('overlay');
const ctx = canvas.getContext('2d');
const gestureNameInput = document.getElementById('gesture-name');
const gestureTypeSelect = document.getElementById('gesture-type');
const sampleList = document.getElementById('sample-list');
const recordBtn = document.getElementById('record-btn');
const saveBtn = document.getElementById('save-btn');

// Initialize camera
async function initCamera() {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({
            video: { width: 640, height: 480 }
        });
        video.srcObject = stream;
        video.onloadedmetadata = () => {
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;
            recordBtn.disabled = false;
        };
    } catch (err) {
        console.error('Camera error:', err);
        alert('Could not access camera. Please grant permission.');
    }
}

// Connect to landmarks WebSocket
function connectWebSocket() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${location.host}/api/landmarks`);

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        currentLandmarks = data.hands;
        drawLandmarks(data.hands);

        if (recording && gestureTypeSelect.value === 'dynamic') {
            // Buffer path points for dynamic gestures
            if (data.hands.length > 0) {
                pathBuffer.push({
                    x: data.hands[0].points[0].x, // Wrist
                    y: data.hands[0].points[0].y,
                    timestamp: data.timestamp
                });
            }
        }
    };

    ws.onclose = () => {
        setTimeout(connectWebSocket, 1000);
    };
}

// Draw hand landmarks on canvas
function drawLandmarks(hands) {
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const hand of hands) {
        // Draw connections
        ctx.strokeStyle = '#e94560';
        ctx.lineWidth = 2;

        const connections = [
            [0, 1], [1, 2], [2, 3], [3, 4],      // Thumb
            [0, 5], [5, 6], [6, 7], [7, 8],      // Index
            [0, 9], [9, 10], [10, 11], [11, 12], // Middle
            [0, 13], [13, 14], [14, 15], [15, 16], // Ring
            [0, 17], [17, 18], [18, 19], [19, 20], // Pinky
            [5, 9], [9, 13], [13, 17]            // Palm
        ];

        for (const [i, j] of connections) {
            const p1 = hand.points[i];
            const p2 = hand.points[j];
            ctx.beginPath();
            ctx.moveTo(p1.x * canvas.width, p1.y * canvas.height);
            ctx.lineTo(p2.x * canvas.width, p2.y * canvas.height);
            ctx.stroke();
        }

        // Draw points
        ctx.fillStyle = '#fff';
        for (const point of hand.points) {
            ctx.beginPath();
            ctx.arc(point.x * canvas.width, point.y * canvas.height, 4, 0, Math.PI * 2);
            ctx.fill();
        }
    }
}

// Record a sample
function recordSample() {
    const type = gestureTypeSelect.value;

    if (type === 'static') {
        // Capture current landmarks
        if (!currentLandmarks || currentLandmarks.length === 0) {
            alert('No hand detected. Please position your hand in view.');
            return;
        }

        samples.push({
            type: 'static',
            landmarks: currentLandmarks[0].points,
            timestamp: Date.now()
        });

        updateSampleList();
    } else {
        // Record dynamic gesture (2 seconds)
        recording = true;
        pathBuffer = [];
        recordBtn.textContent = 'Recording...';
        recordBtn.classList.add('recording');
        recordBtn.disabled = true;

        setTimeout(() => {
            recording = false;
            recordBtn.textContent = 'Record Sample';
            recordBtn.classList.remove('recording');
            recordBtn.disabled = false;

            if (pathBuffer.length < 10) {
                alert('Not enough movement detected. Please try again.');
                return;
            }

            samples.push({
                type: 'dynamic',
                path: pathBuffer,
                timestamp: Date.now()
            });

            updateSampleList();
        }, 2000);
    }
}

// Update sample list UI
function updateSampleList() {
    const count = samples.length;
    document.querySelector('h3').textContent = `Samples (${count}/5)`;

    if (count === 0) {
        sampleList.innerHTML = '<p class="text-muted">No samples recorded yet</p>';
    } else {
        sampleList.innerHTML = samples.map((s, i) => `
            <div class="sample-item">
                <span>Sample ${i + 1} (${s.type})</span>
                <button onclick="removeSample(${i})" style="margin-left: auto;">×</button>
            </div>
        `).join('');
    }

    saveBtn.disabled = count < 3;
}

// Remove a sample
function removeSample(index) {
    samples.splice(index, 1);
    updateSampleList();
}

// Save gesture
async function saveGesture() {
    const name = gestureNameInput.value.trim();
    if (!name) {
        alert('Please enter a gesture name.');
        return;
    }

    const type = gestureTypeSelect.value;

    try {
        // Create gesture
        const response = await fetch(`${API_BASE}/gestures`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, type })
        });

        if (!response.ok) {
            throw new Error('Failed to create gesture');
        }

        const gesture = await response.json();

        // Save samples
        await fetch(`${API_BASE}/gestures/${gesture.id}/samples`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ samples })
        });

        alert('Gesture saved successfully!');
        window.location.href = '/';
    } catch (err) {
        console.error('Save error:', err);
        alert('Failed to save gesture: ' + err.message);
    }
}

// Type change handler
gestureTypeSelect.onchange = () => {
    const isStatic = gestureTypeSelect.value === 'static';
    document.getElementById('static-instructions').style.display = isStatic ? 'block' : 'none';
    document.getElementById('dynamic-instructions').style.display = isStatic ? 'none' : 'block';
    samples = [];
    updateSampleList();
};

// Event listeners
recordBtn.onclick = recordSample;
saveBtn.onclick = saveGesture;

// Initialize
initCamera();
connectWebSocket();
```

**Step 2: Commit**

```bash
git add web/js/record.js
git commit -m "feat: add gesture recording JavaScript with landmark visualization"
```

---

## Phase 3: Sample Storage API

### Task 5: Add Sample Storage Endpoints

**Files:**
- Create: `internal/store/sample.go`
- Create: `internal/server/api/samples.go`
- Modify: `internal/store/migrations.go`

**Step 1: Add samples table to migrations**

```sql
CREATE TABLE IF NOT EXISTS gesture_samples (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
    sample_index INTEGER NOT NULL,
    data TEXT NOT NULL,  -- JSON encoded landmarks or path
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gesture_samples_gesture_id ON gesture_samples(gesture_id);
```

**Step 2: Write sample repository**

```go
// internal/store/sample.go
package store

import (
    "encoding/json"
)

type Sample struct {
    ID          int64
    GestureID   string
    SampleIndex int
    Data        json.RawMessage
    CreatedAt   time.Time
}

type SampleRepository struct {
    db *sql.DB
}

func (s *Store) Samples() *SampleRepository {
    return &SampleRepository{db: s.db}
}

func (r *SampleRepository) Create(gestureID string, samples []json.RawMessage) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare(`INSERT INTO gesture_samples (gesture_id, sample_index, data) VALUES (?, ?, ?)`)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for i, data := range samples {
        if _, err := stmt.Exec(gestureID, i, string(data)); err != nil {
            return err
        }
    }

    // Update sample count
    _, err = tx.Exec(`UPDATE gestures SET samples = ? WHERE id = ?`, len(samples), gestureID)
    if err != nil {
        return err
    }

    return tx.Commit()
}

func (r *SampleRepository) GetByGestureID(gestureID string) ([]Sample, error) {
    rows, err := r.db.Query(
        `SELECT id, gesture_id, sample_index, data, created_at FROM gesture_samples WHERE gesture_id = ? ORDER BY sample_index`,
        gestureID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var samples []Sample
    for rows.Next() {
        var s Sample
        if err := rows.Scan(&s.ID, &s.GestureID, &s.SampleIndex, &s.Data, &s.CreatedAt); err != nil {
            return nil, err
        }
        samples = append(samples, s)
    }
    return samples, rows.Err()
}
```

**Step 3: Write samples API handler**

```go
// internal/server/api/samples.go
package api

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/ayusman/kuchipudi/internal/store"
)

type SamplesHandler struct {
    store *store.Store
}

func NewSamplesHandler(s *store.Store) *SamplesHandler {
    return &SamplesHandler{store: s}
}

func (h *SamplesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Parse gesture ID from path: /api/gestures/{id}/samples
    path := strings.TrimPrefix(r.URL.Path, "/api/gestures/")
    parts := strings.Split(path, "/")
    if len(parts) != 2 || parts[1] != "samples" {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    gestureID := parts[0]

    switch r.Method {
    case http.MethodGet:
        h.list(w, r, gestureID)
    case http.MethodPost:
        h.create(w, r, gestureID)
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    }
}

type createSamplesRequest struct {
    Samples []json.RawMessage `json:"samples"`
}

func (h *SamplesHandler) create(w http.ResponseWriter, r *http.Request, gestureID string) {
    var req createSamplesRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    if err := h.store.Samples().Create(gestureID, req.Samples); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *SamplesHandler) list(w http.ResponseWriter, r *http.Request, gestureID string) {
    samples, err := h.store.Samples().GetByGestureID(gestureID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"samples": samples})
}
```

**Step 4: Commit**

```bash
git add internal/store/sample.go internal/store/migrations.go internal/server/api/samples.go
git commit -m "feat: add sample storage and API endpoints"
```

---

### Task 6: Process Samples into Templates

**Files:**
- Create: `internal/gesture/trainer.go`
- Create: `internal/gesture/trainer_test.go`

**Step 1: Write gesture trainer**

```go
// internal/gesture/trainer.go
package gesture

import (
    "encoding/json"

    "github.com/ayusman/kuchipudi/internal/detector"
)

// Trainer processes recorded samples into gesture templates
type Trainer struct{}

func NewTrainer() *Trainer {
    return &Trainer{}
}

// TrainStatic averages multiple landmark samples into a single template
func (t *Trainer) TrainStatic(samples []json.RawMessage) ([]detector.Point3D, error) {
    if len(samples) == 0 {
        return nil, fmt.Errorf("no samples provided")
    }

    // Parse all samples
    var allLandmarks [][]detector.Point3D
    for _, raw := range samples {
        var sample struct {
            Landmarks []detector.Point3D `json:"landmarks"`
        }
        if err := json.Unmarshal(raw, &sample); err != nil {
            return nil, err
        }
        allLandmarks = append(allLandmarks, sample.Landmarks)
    }

    // Average landmarks
    numPoints := len(allLandmarks[0])
    averaged := make([]detector.Point3D, numPoints)

    for i := 0; i < numPoints; i++ {
        var sumX, sumY, sumZ float64
        for _, landmarks := range allLandmarks {
            sumX += landmarks[i].X
            sumY += landmarks[i].Y
            sumZ += landmarks[i].Z
        }
        n := float64(len(allLandmarks))
        averaged[i] = detector.Point3D{
            X: sumX / n,
            Y: sumY / n,
            Z: sumZ / n,
        }
    }

    return averaged, nil
}

// TrainDynamic averages multiple path samples using DTW alignment
func (t *Trainer) TrainDynamic(samples []json.RawMessage) ([]PathPoint, error) {
    if len(samples) == 0 {
        return nil, fmt.Errorf("no samples provided")
    }

    // Parse all samples
    var allPaths [][]PathPoint
    for _, raw := range samples {
        var sample struct {
            Path []PathPoint `json:"path"`
        }
        if err := json.Unmarshal(raw, &sample); err != nil {
            return nil, err
        }
        allPaths = append(allPaths, sample.Path)
    }

    // Use first path as reference, resample others to match length
    reference := normalizePath(allPaths[0])

    // Average all paths (simplified - could use DTW barycenter for better results)
    averaged := make([]PathPoint, len(reference))
    for i := range reference {
        var sumX, sumY float64
        for _, path := range allPaths {
            normalized := normalizePath(path)
            // Resample to match reference length
            idx := i * len(normalized) / len(reference)
            if idx >= len(normalized) {
                idx = len(normalized) - 1
            }
            sumX += normalized[idx].X
            sumY += normalized[idx].Y
        }
        n := float64(len(allPaths))
        averaged[i] = PathPoint{
            X:         sumX / n,
            Y:         sumY / n,
            Timestamp: reference[i].Timestamp,
        }
    }

    return averaged, nil
}
```

**Step 2: Write tests**

```go
// internal/gesture/trainer_test.go
package gesture

import (
    "encoding/json"
    "testing"
)

func TestTrainer_TrainStatic(t *testing.T) {
    trainer := NewTrainer()

    samples := []json.RawMessage{
        json.RawMessage(`{"landmarks": [{"x": 0.5, "y": 0.5, "z": 0}]}`),
        json.RawMessage(`{"landmarks": [{"x": 0.6, "y": 0.4, "z": 0}]}`),
    }

    result, err := trainer.TrainStatic(samples)
    if err != nil {
        t.Fatalf("TrainStatic() error = %v", err)
    }

    if len(result) != 1 {
        t.Fatalf("expected 1 landmark, got %d", len(result))
    }

    // Average should be (0.55, 0.45, 0)
    if result[0].X != 0.55 || result[0].Y != 0.45 {
        t.Errorf("wrong average: got (%f, %f)", result[0].X, result[0].Y)
    }
}
```

**Step 3: Commit**

```bash
git add internal/gesture/trainer.go internal/gesture/trainer_test.go
git commit -m "feat: add gesture trainer for processing samples into templates"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Add MJPEG camera stream endpoint |
| 2 | Add WebSocket endpoint for real-time landmarks |
| 3 | Create recording page HTML |
| 4 | Create recording JavaScript with landmark visualization |
| 5 | Add sample storage API and database tables |
| 6 | Add gesture trainer for processing samples |

**User Flow:**
1. Navigate to /record.html
2. Enter gesture name and select type (static/dynamic)
3. Record 3-5 samples
4. Click Save to store gesture

**Technical Notes:**
- WebRTC for browser camera access (fallback to MJPEG stream)
- Canvas overlay for landmark visualization
- WebSocket for low-latency landmark updates
- SQLite stores raw samples; trainer processes into templates on save
