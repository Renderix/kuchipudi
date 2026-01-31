# MediaPipe Hand Detection Integration Plan

> **For Claude:** Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the mock hand detector with real MediaPipe hand detection for accurate 21-landmark hand tracking.

**Architecture:** Use MediaPipe's hand landmarker via a Python subprocess or CGO bindings. The detector runs on-demand when motion is detected to minimize CPU usage.

**Tech Stack:** MediaPipe (Python or C++), CGO (if C++ route), subprocess IPC (if Python route)

---

## Phase 1: Evaluate Integration Approaches

### Task 1: Research MediaPipe Options

**Files:**
- Create: `docs/research/mediapipe-options.md`

**Step 1: Document available options**

| Approach | Pros | Cons |
|----------|------|------|
| Python subprocess | Easy setup, official SDK | Process startup overhead, IPC latency |
| MediaPipe C++ via CGO | Low latency, native | Complex build, CGO dependency |
| ONNX Runtime | Pure Go possible | Need to export model, less accurate |
| TensorFlow Lite Go | Official bindings | Limited hand model support |

**Step 2: Benchmark Python subprocess overhead**

Create a simple Python script that loads MediaPipe and processes one frame:
```python
import mediapipe as mp
import sys
import json
import numpy as np

mp_hands = mp.solutions.hands
hands = mp_hands.Hands(static_image_mode=True, max_num_hands=2)

# Read image from stdin, process, output JSON
```

Measure cold start time and per-frame latency.

**Step 3: Document recommendation**

Recommend Python subprocess for V1 (simpler), with option to migrate to C++ later.

**Step 4: Commit**

```bash
git add docs/research/
git commit -m "docs: research MediaPipe integration options"
```

---

## Phase 2: Python MediaPipe Service

### Task 2: Create MediaPipe Python Service

**Files:**
- Create: `scripts/mediapipe_service.py`
- Create: `scripts/requirements.txt`

**Step 1: Write requirements.txt**

```
mediapipe>=0.10.0
numpy>=1.24.0
opencv-python>=4.8.0
```

**Step 2: Write mediapipe_service.py**

```python
#!/usr/bin/env python3
"""
MediaPipe hand detection service.
Reads JPEG frames from stdin, outputs JSON landmarks to stdout.

Protocol:
- Input: 4-byte length (big-endian) + JPEG bytes
- Output: JSON line with landmarks array

Run: python3 mediapipe_service.py
"""

import sys
import struct
import json
import cv2
import numpy as np
import mediapipe as mp

mp_hands = mp.solutions.hands

def main():
    hands = mp_hands.Hands(
        static_image_mode=False,
        max_num_hands=2,
        min_detection_confidence=0.5,
        min_tracking_confidence=0.5
    )

    while True:
        # Read frame length (4 bytes, big-endian)
        length_bytes = sys.stdin.buffer.read(4)
        if len(length_bytes) < 4:
            break

        length = struct.unpack('>I', length_bytes)[0]

        # Read JPEG data
        jpeg_data = sys.stdin.buffer.read(length)
        if len(jpeg_data) < length:
            break

        # Decode image
        nparr = np.frombuffer(jpeg_data, np.uint8)
        image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        if image is None:
            print(json.dumps({"hands": []}), flush=True)
            continue

        # Convert BGR to RGB
        image_rgb = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)

        # Process
        results = hands.process(image_rgb)

        # Format output
        output = {"hands": []}

        if results.multi_hand_landmarks:
            for i, hand_landmarks in enumerate(results.multi_hand_landmarks):
                handedness = "Right"
                if results.multi_handedness:
                    handedness = results.multi_handedness[i].classification[0].label

                points = []
                for lm in hand_landmarks.landmark:
                    points.append({
                        "x": lm.x,
                        "y": lm.y,
                        "z": lm.z
                    })

                output["hands"].append({
                    "points": points,
                    "handedness": handedness,
                    "score": results.multi_handedness[i].classification[0].score if results.multi_handedness else 0.9
                })

        print(json.dumps(output), flush=True)

    hands.close()

if __name__ == "__main__":
    main()
```

**Step 3: Test the service**

```bash
cd scripts
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
# Test with a sample image
```

**Step 4: Commit**

```bash
git add scripts/
git commit -m "feat: add MediaPipe Python service for hand detection"
```

---

### Task 3: Create Go MediaPipe Detector

**Files:**
- Create: `internal/detector/mediapipe.go`
- Create: `internal/detector/mediapipe_test.go`

**Step 1: Write the failing test**

```go
// internal/detector/mediapipe_test.go
package detector

import (
    "os/exec"
    "testing"
)

func TestMediaPipeDetector_PythonAvailable(t *testing.T) {
    _, err := exec.LookPath("python3")
    if err != nil {
        t.Skip("python3 not available")
    }
}

func TestMediaPipeDetector_Detect(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    d, err := NewMediaPipeDetector(DefaultConfig())
    if err != nil {
        t.Skipf("MediaPipe not available: %v", err)
    }
    defer d.Close()

    // Create a test frame (black image)
    // ... test detection returns empty or valid hands
}
```

**Step 2: Write MediaPipe detector**

```go
// internal/detector/mediapipe.go
package detector

import (
    "bufio"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "sync"

    "gocv.io/x/gocv"
)

type MediaPipeDetector struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout *bufio.Reader
    mu     sync.Mutex
}

func NewMediaPipeDetector(config Config) (*MediaPipeDetector, error) {
    // Find the Python script
    scriptPath := findMediaPipeScript()
    if scriptPath == "" {
        return nil, fmt.Errorf("mediapipe_service.py not found")
    }

    cmd := exec.Command("python3", scriptPath)

    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, fmt.Errorf("create stdin pipe: %w", err)
    }

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, fmt.Errorf("create stdout pipe: %w", err)
    }

    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("start mediapipe service: %w", err)
    }

    return &MediaPipeDetector{
        cmd:    cmd,
        stdin:  stdin,
        stdout: bufio.NewReader(stdout),
    }, nil
}

func (d *MediaPipeDetector) Detect(frame *gocv.Mat) ([]HandLandmarks, error) {
    d.mu.Lock()
    defer d.mu.Unlock()

    // Encode frame as JPEG
    buf, err := gocv.IMEncode(".jpg", *frame)
    if err != nil {
        return nil, fmt.Errorf("encode frame: %w", err)
    }
    defer buf.Close()

    data := buf.GetBytes()

    // Write length + data
    length := make([]byte, 4)
    binary.BigEndian.PutUint32(length, uint32(len(data)))

    if _, err := d.stdin.Write(length); err != nil {
        return nil, fmt.Errorf("write length: %w", err)
    }
    if _, err := d.stdin.Write(data); err != nil {
        return nil, fmt.Errorf("write data: %w", err)
    }

    // Read response
    line, err := d.stdout.ReadString('\n')
    if err != nil {
        return nil, fmt.Errorf("read response: %w", err)
    }

    var response struct {
        Hands []HandLandmarks `json:"hands"`
    }
    if err := json.Unmarshal([]byte(line), &response); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }

    return response.Hands, nil
}

func (d *MediaPipeDetector) Close() error {
    d.stdin.Close()
    return d.cmd.Wait()
}

func findMediaPipeScript() string {
    candidates := []string{
        "scripts/mediapipe_service.py",
        "../scripts/mediapipe_service.py",
        filepath.Join(os.Getenv("HOME"), ".kuchipudi/scripts/mediapipe_service.py"),
    }
    for _, path := range candidates {
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }
    return ""
}
```

**Step 3: Run tests**

```bash
go test ./internal/detector/... -v
```

**Step 4: Commit**

```bash
git add internal/detector/mediapipe.go internal/detector/mediapipe_test.go
git commit -m "feat: add MediaPipe detector with Python subprocess"
```

---

### Task 4: Wire MediaPipe to App

**Files:**
- Modify: `internal/app/app.go`
- Modify: `cmd/kuchipudi/main.go`

**Step 1: Update App to use MediaPipe by default**

In `app.go`, change New() to try MediaPipe first, fall back to mock:

```go
func New(config Config) *App {
    a := &App{
        // ... existing initialization
    }

    // Try MediaPipe first
    if mp, err := detector.NewMediaPipeDetector(detector.DefaultConfig()); err == nil {
        a.detector = mp
        log.Println("Using MediaPipe hand detection")
    } else {
        log.Printf("MediaPipe not available (%v), using mock detector", err)
        a.detector = detector.NewMockDetector()
    }

    return a
}
```

**Step 2: Add graceful detector switching**

```go
func (a *App) SetDetector(d detector.Detector) {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.detector != nil {
        a.detector.Close()
    }
    a.detector = d
}
```

**Step 3: Verify build**

```bash
go build ./cmd/kuchipudi
```

**Step 4: Commit**

```bash
git add internal/app/app.go cmd/kuchipudi/main.go
git commit -m "feat: wire MediaPipe detector to app with fallback"
```

---

## Phase 3: Performance Optimization

### Task 5: Add Lazy Loading

**Files:**
- Modify: `internal/detector/mediapipe.go`

**Step 1: Implement lazy initialization**

Only start the Python process when first detection is requested:

```go
type MediaPipeDetector struct {
    config Config
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout *bufio.Reader
    mu     sync.Mutex
    started bool
}

func (d *MediaPipeDetector) ensureStarted() error {
    if d.started {
        return nil
    }
    // Start process here
    d.started = true
    return nil
}

func (d *MediaPipeDetector) Detect(frame *gocv.Mat) ([]HandLandmarks, error) {
    d.mu.Lock()
    defer d.mu.Unlock()

    if err := d.ensureStarted(); err != nil {
        return nil, err
    }
    // ... rest of detection
}
```

**Step 2: Add idle shutdown**

Shut down the Python process after 30 seconds of inactivity:

```go
type MediaPipeDetector struct {
    // ... existing fields
    lastUsed  time.Time
    idleTimer *time.Timer
}

func (d *MediaPipeDetector) Detect(...) {
    // ... detection code
    d.lastUsed = time.Now()
    d.resetIdleTimer()
}

func (d *MediaPipeDetector) resetIdleTimer() {
    if d.idleTimer != nil {
        d.idleTimer.Stop()
    }
    d.idleTimer = time.AfterFunc(30*time.Second, func() {
        d.shutdown()
    })
}
```

**Step 3: Commit**

```bash
git add internal/detector/mediapipe.go
git commit -m "perf: add lazy loading and idle shutdown to MediaPipe detector"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Research and document MediaPipe integration options |
| 2 | Create Python MediaPipe service with stdin/stdout protocol |
| 3 | Create Go detector that wraps Python service |
| 4 | Wire MediaPipe to main app with mock fallback |
| 5 | Add lazy loading and idle shutdown for efficiency |

**Prerequisites:**
- Python 3.8+ installed
- `pip install mediapipe opencv-python numpy`

**Expected Performance:**
- Cold start: ~2-3 seconds (MediaPipe model loading)
- Per-frame: ~30-50ms on M1 Mac
- Idle: 0% CPU (process shut down)
