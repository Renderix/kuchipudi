# Kuchipudi V1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a macOS hand gesture recognition daemon that detects gestures from webcam and triggers configurable actions via plugins.

**Architecture:** Hybrid detection system (motion detection triggers ML) keeps CPU <1% when idle. Plugin-based action system allows extensibility. SQLite stores gesture templates and mappings. Tray app + browser UI for configuration.

**Tech Stack:** Go 1.21+, GoCV (OpenCV bindings), MediaPipe (hand detection), SQLite (modernc.org/sqlite), systray (getlantern/systray), vanilla HTML/CSS/JS for web UI.

---

## Phase 1: Project Foundation

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `cmd/kuchipudi/main.go`

**Step 1: Initialize Go module**

Run: `go mod init github.com/ayusman/kuchipudi`
Expected: Creates go.mod file

**Step 2: Create minimal main.go**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Kuchipudi - Hand Gesture Recognition")
	os.Exit(0)
}
```

**Step 3: Build and run to verify**

Run: `go build -o bin/kuchipudi ./cmd/kuchipudi && ./bin/kuchipudi`
Expected: Prints "Kuchipudi - Hand Gesture Recognition"

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "feat: initialize go module with minimal main"
```

---

### Task 2: Setup SQLite Store

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`
- Create: `internal/store/migrations.go`

**Step 1: Add SQLite dependency**

Run: `go get modernc.org/sqlite`
Expected: Adds sqlite to go.mod

**Step 2: Write the failing test**

```go
// internal/store/store_test.go
package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore_CreatesDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestNewStore_RunsMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Verify tables exist
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='gestures'").Scan(&count)
	if err != nil {
		t.Fatalf("query error = %v", err)
	}
	if count != 1 {
		t.Error("gestures table was not created")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/store/... -v`
Expected: FAIL - package not found

**Step 4: Write store implementation**

```go
// internal/store/store.go
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db   *sql.DB
	path string
}

func New(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &Store{db: db, path: dbPath}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}
```

**Step 5: Write migrations**

```go
// internal/store/migrations.go
package store

const schema = `
CREATE TABLE IF NOT EXISTS gestures (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	type TEXT NOT NULL CHECK(type IN ('static', 'dynamic')),
	tolerance REAL NOT NULL DEFAULT 0.15,
	samples INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS gesture_landmarks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
	landmark_index INTEGER NOT NULL,
	x REAL NOT NULL,
	y REAL NOT NULL,
	z REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS gesture_paths (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
	sequence INTEGER NOT NULL,
	x REAL NOT NULL,
	y REAL NOT NULL,
	timestamp_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS actions (
	id TEXT PRIMARY KEY,
	gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
	plugin_name TEXT NOT NULL,
	action_name TEXT NOT NULL,
	config TEXT NOT NULL DEFAULT '{}',
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gesture_landmarks_gesture_id ON gesture_landmarks(gesture_id);
CREATE INDEX IF NOT EXISTS idx_gesture_paths_gesture_id ON gesture_paths(gesture_id);
CREATE INDEX IF NOT EXISTS idx_actions_gesture_id ON actions(gesture_id);
`

func (s *Store) migrate() error {
	_, err := s.db.Exec(schema)
	return err
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/store/... -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/store/ go.mod go.sum
git commit -m "feat: add SQLite store with schema migrations"
```

---

### Task 3: Gesture Repository

**Files:**
- Create: `internal/store/gesture.go`
- Create: `internal/store/gesture_test.go`

**Step 1: Write the failing test**

```go
// internal/store/gesture_test.go
package store

import (
	"path/filepath"
	"testing"
)

func TestGestureRepository_Create(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	g := &Gesture{
		ID:        "test-1",
		Name:      "thumbs-up",
		Type:      GestureTypeStatic,
		Tolerance: 0.15,
	}

	err := s.Gestures().Create(g)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := s.Gestures().GetByID("test-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "thumbs-up" {
		t.Errorf("Name = %q, want %q", got.Name, "thumbs-up")
	}
}

func TestGestureRepository_List(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	s.Gestures().Create(&Gesture{ID: "g1", Name: "swipe-left", Type: GestureTypeDynamic})
	s.Gestures().Create(&Gesture{ID: "g2", Name: "swipe-right", Type: GestureTypeDynamic})

	gestures, err := s.Gestures().List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(gestures) != 2 {
		t.Errorf("len(gestures) = %d, want 2", len(gestures))
	}
}

func TestGestureRepository_Delete(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	s.Gestures().Create(&Gesture{ID: "g1", Name: "test", Type: GestureTypeStatic})

	err := s.Gestures().Delete("g1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = s.Gestures().GetByID("g1")
	if err != ErrNotFound {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return s
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/... -v`
Expected: FAIL - Gesture type not defined

**Step 3: Write gesture repository**

```go
// internal/store/gesture.go
package store

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type GestureType string

const (
	GestureTypeStatic  GestureType = "static"
	GestureTypeDynamic GestureType = "dynamic"
)

type Gesture struct {
	ID        string
	Name      string
	Type      GestureType
	Tolerance float64
	Samples   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GestureRepository struct {
	db *sql.DB
}

func (s *Store) Gestures() *GestureRepository {
	return &GestureRepository{db: s.db}
}

func (r *GestureRepository) Create(g *Gesture) error {
	_, err := r.db.Exec(
		`INSERT INTO gestures (id, name, type, tolerance, samples) VALUES (?, ?, ?, ?, ?)`,
		g.ID, g.Name, g.Type, g.Tolerance, g.Samples,
	)
	return err
}

func (r *GestureRepository) GetByID(id string) (*Gesture, error) {
	g := &Gesture{}
	err := r.db.QueryRow(
		`SELECT id, name, type, tolerance, samples, created_at, updated_at FROM gestures WHERE id = ?`,
		id,
	).Scan(&g.ID, &g.Name, &g.Type, &g.Tolerance, &g.Samples, &g.CreatedAt, &g.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (r *GestureRepository) GetByName(name string) (*Gesture, error) {
	g := &Gesture{}
	err := r.db.QueryRow(
		`SELECT id, name, type, tolerance, samples, created_at, updated_at FROM gestures WHERE name = ?`,
		name,
	).Scan(&g.ID, &g.Name, &g.Type, &g.Tolerance, &g.Samples, &g.CreatedAt, &g.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (r *GestureRepository) List() ([]*Gesture, error) {
	rows, err := r.db.Query(
		`SELECT id, name, type, tolerance, samples, created_at, updated_at FROM gestures ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gestures []*Gesture
	for rows.Next() {
		g := &Gesture{}
		if err := rows.Scan(&g.ID, &g.Name, &g.Type, &g.Tolerance, &g.Samples, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		gestures = append(gestures, g)
	}
	return gestures, rows.Err()
}

func (r *GestureRepository) Update(g *Gesture) error {
	result, err := r.db.Exec(
		`UPDATE gestures SET name = ?, type = ?, tolerance = ?, samples = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		g.Name, g.Type, g.Tolerance, g.Samples, g.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GestureRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM gestures WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/gesture.go internal/store/gesture_test.go
git commit -m "feat: add gesture repository with CRUD operations"
```

---

## Phase 2: Camera Capture & Motion Detection

### Task 4: Camera Capture with GoCV

**Files:**
- Create: `internal/capture/camera.go`
- Create: `internal/capture/camera_test.go`

**Step 1: Add GoCV dependency**

Note: GoCV requires OpenCV to be installed. On macOS:
```bash
brew install opencv
```

Then:
Run: `go get -u gocv.io/x/gocv`
Expected: Adds gocv to go.mod

**Step 2: Write camera interface and implementation**

```go
// internal/capture/camera.go
package capture

import (
	"fmt"
	"image"
	"sync"

	"gocv.io/x/gocv"
)

type Frame struct {
	Image     image.Image
	Timestamp int64
	Width     int
	Height    int
}

type Camera struct {
	deviceID int
	capture  *gocv.VideoCapture
	mu       sync.Mutex
	running  bool
	fps      int
}

func NewCamera(deviceID int) *Camera {
	return &Camera{
		deviceID: deviceID,
		fps:      5, // Start with low FPS for idle mode
	}
}

func (c *Camera) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capture != nil {
		return nil // Already open
	}

	cap, err := gocv.OpenVideoCapture(c.deviceID)
	if err != nil {
		return fmt.Errorf("open camera %d: %w", c.deviceID, err)
	}

	// Set resolution to 640x480 for performance
	cap.Set(gocv.VideoCaptureFrameWidth, 640)
	cap.Set(gocv.VideoCaptureFrameHeight, 480)

	c.capture = cap
	c.running = true
	return nil
}

func (c *Camera) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.running = false
	if c.capture != nil {
		err := c.capture.Close()
		c.capture = nil
		return err
	}
	return nil
}

func (c *Camera) ReadFrame() (*gocv.Mat, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capture == nil || !c.running {
		return nil, fmt.Errorf("camera not open")
	}

	mat := gocv.NewMat()
	if ok := c.capture.Read(&mat); !ok {
		mat.Close()
		return nil, fmt.Errorf("failed to read frame")
	}

	if mat.Empty() {
		mat.Close()
		return nil, fmt.Errorf("empty frame")
	}

	return &mat, nil
}

func (c *Camera) SetFPS(fps int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fps = fps
}

func (c *Camera) FPS() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fps
}

func (c *Camera) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.capture != nil && c.running
}
```

**Step 3: Write basic test (will need actual camera to fully test)**

```go
// internal/capture/camera_test.go
package capture

import (
	"testing"
)

func TestNewCamera(t *testing.T) {
	cam := NewCamera(0)
	if cam == nil {
		t.Fatal("NewCamera returned nil")
	}
	if cam.deviceID != 0 {
		t.Errorf("deviceID = %d, want 0", cam.deviceID)
	}
	if cam.fps != 5 {
		t.Errorf("fps = %d, want 5", cam.fps)
	}
}

func TestCamera_SetFPS(t *testing.T) {
	cam := NewCamera(0)
	cam.SetFPS(15)
	if got := cam.FPS(); got != 15 {
		t.Errorf("FPS() = %d, want 15", got)
	}
}

// Integration test - requires actual camera
func TestCamera_OpenClose_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cam := NewCamera(0)
	if err := cam.Open(); err != nil {
		t.Skipf("no camera available: %v", err)
	}
	defer cam.Close()

	if !cam.IsOpen() {
		t.Error("camera should be open")
	}

	frame, err := cam.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	defer frame.Close()

	if frame.Empty() {
		t.Error("frame should not be empty")
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/capture/... -v -short`
Expected: PASS (unit tests only)

**Step 5: Commit**

```bash
git add internal/capture/ go.mod go.sum
git commit -m "feat: add camera capture with GoCV"
```

---

### Task 5: Motion Detection

**Files:**
- Create: `internal/capture/motion.go`
- Create: `internal/capture/motion_test.go`

**Step 1: Write the failing test**

```go
// internal/capture/motion_test.go
package capture

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestMotionDetector_NoMotion(t *testing.T) {
	md := NewMotionDetector(0.05) // 5% threshold

	// Create two identical frames
	frame1 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame1.Close()
	frame2 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame2.Close()

	// First frame sets baseline
	md.Detect(&frame1)

	// Second identical frame should not detect motion
	detected, _ := md.Detect(&frame2)
	if detected {
		t.Error("motion detected on identical frames")
	}
}

func TestMotionDetector_WithMotion(t *testing.T) {
	md := NewMotionDetector(0.01) // 1% threshold for test sensitivity

	// Create first frame (black)
	frame1 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame1.Close()

	// Create second frame (white) - significant difference
	frame2 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame2.Close()
	frame2.SetTo(gocv.NewScalar(255, 255, 255, 0))

	// First frame sets baseline
	md.Detect(&frame1)

	// Second different frame should detect motion
	detected, score := md.Detect(&frame2)
	if !detected {
		t.Errorf("motion not detected, score = %f", score)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/capture/... -v -run Motion`
Expected: FAIL - MotionDetector not defined

**Step 3: Write motion detector**

```go
// internal/capture/motion.go
package capture

import (
	"gocv.io/x/gocv"
)

type MotionDetector struct {
	threshold    float64
	prevGray     gocv.Mat
	initialized  bool
}

func NewMotionDetector(threshold float64) *MotionDetector {
	return &MotionDetector{
		threshold: threshold,
		prevGray:  gocv.NewMat(),
	}
}

func (m *MotionDetector) Detect(frame *gocv.Mat) (bool, float64) {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(*frame, &gray, gocv.ColorBGRToGray)

	// Apply Gaussian blur to reduce noise
	gocv.GaussianBlur(gray, &gray, image.Point{X: 21, Y: 21}, 0, 0, gocv.BorderDefault)

	if !m.initialized {
		gray.CopyTo(&m.prevGray)
		m.initialized = true
		return false, 0
	}

	// Calculate absolute difference
	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(gray, m.prevGray, &diff)

	// Threshold the difference
	gocv.Threshold(diff, &diff, 25, 255, gocv.ThresholdBinary)

	// Calculate percentage of changed pixels
	nonZero := gocv.CountNonZero(diff)
	totalPixels := diff.Rows() * diff.Cols()
	changePercent := float64(nonZero) / float64(totalPixels)

	// Update previous frame
	gray.CopyTo(&m.prevGray)

	return changePercent > m.threshold, changePercent
}

func (m *MotionDetector) Reset() {
	m.initialized = false
	if !m.prevGray.Empty() {
		m.prevGray.Close()
		m.prevGray = gocv.NewMat()
	}
}

func (m *MotionDetector) Close() {
	if !m.prevGray.Empty() {
		m.prevGray.Close()
	}
}

func (m *MotionDetector) SetThreshold(threshold float64) {
	m.threshold = threshold
}
```

**Step 4: Add missing import**

Add to motion.go imports:
```go
import (
	"image"

	"gocv.io/x/gocv"
)
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/capture/... -v -run Motion`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/capture/motion.go internal/capture/motion_test.go
git commit -m "feat: add motion detection with frame differencing"
```

---

## Phase 3: Hand Detection with MediaPipe

### Task 6: MediaPipe Hand Detector Interface

**Files:**
- Create: `internal/detector/detector.go`
- Create: `internal/detector/landmarks.go`

Note: Full MediaPipe integration requires CGO and the MediaPipe C++ library. For now, we'll create the interface and a mock implementation. Real MediaPipe integration can be swapped in later.

**Step 1: Write landmark types**

```go
// internal/detector/landmarks.go
package detector

// HandLandmark indices based on MediaPipe hand model
const (
	Wrist             = 0
	ThumbCMC          = 1
	ThumbMCP          = 2
	ThumbIP           = 3
	ThumbTip          = 4
	IndexMCP          = 5
	IndexPIP          = 6
	IndexDIP          = 7
	IndexTip          = 8
	MiddleMCP         = 9
	MiddlePIP         = 10
	MiddleDIP         = 11
	MiddleTip         = 12
	RingMCP           = 13
	RingPIP           = 14
	RingDIP           = 15
	RingTip           = 16
	PinkyMCP          = 17
	PinkyPIP          = 18
	PinkyDIP          = 19
	PinkyTip          = 20
	NumLandmarks      = 21
)

type Point3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type HandLandmarks struct {
	Points     [NumLandmarks]Point3D `json:"points"`
	Handedness string                 `json:"handedness"` // "Left" or "Right"
	Score      float64                `json:"score"`      // Detection confidence
}

// Normalize landmarks relative to wrist position and hand size
func (h *HandLandmarks) Normalize() *HandLandmarks {
	normalized := &HandLandmarks{
		Handedness: h.Handedness,
		Score:      h.Score,
	}

	// Use wrist as origin
	wrist := h.Points[Wrist]

	// Calculate hand size (distance from wrist to middle finger MCP)
	middleMCP := h.Points[MiddleMCP]
	handSize := distance3D(wrist, middleMCP)
	if handSize < 0.001 {
		handSize = 1 // Prevent division by zero
	}

	for i, p := range h.Points {
		normalized.Points[i] = Point3D{
			X: (p.X - wrist.X) / handSize,
			Y: (p.Y - wrist.Y) / handSize,
			Z: (p.Z - wrist.Z) / handSize,
		}
	}

	return normalized
}

func distance3D(a, b Point3D) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return sqrt(dx*dx + dy*dy + dz*dz)
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
```

**Step 2: Write detector interface**

```go
// internal/detector/detector.go
package detector

import (
	"gocv.io/x/gocv"
)

// Detector detects hands in video frames
type Detector interface {
	// Detect finds hands in the given frame
	// Returns slice of detected hands (can be empty)
	Detect(frame *gocv.Mat) ([]HandLandmarks, error)

	// Close releases resources
	Close() error
}

// Config holds detector configuration
type Config struct {
	MaxHands       int     // Maximum number of hands to detect (default: 2)
	MinConfidence  float64 // Minimum detection confidence (default: 0.5)
	MinTrackingConf float64 // Minimum tracking confidence (default: 0.5)
}

func DefaultConfig() Config {
	return Config{
		MaxHands:        2,
		MinConfidence:   0.5,
		MinTrackingConf: 0.5,
	}
}
```

**Step 3: Commit**

```bash
git add internal/detector/
git commit -m "feat: add hand detector interface and landmark types"
```

---

### Task 7: Mock Hand Detector (for testing)

**Files:**
- Create: `internal/detector/mock.go`
- Create: `internal/detector/detector_test.go`

**Step 1: Write mock detector**

```go
// internal/detector/mock.go
package detector

import (
	"gocv.io/x/gocv"
)

// MockDetector is a test implementation that returns preset hands
type MockDetector struct {
	hands []HandLandmarks
	err   error
}

func NewMockDetector() *MockDetector {
	return &MockDetector{}
}

func (m *MockDetector) SetHands(hands []HandLandmarks) {
	m.hands = hands
}

func (m *MockDetector) SetError(err error) {
	m.err = err
}

func (m *MockDetector) Detect(frame *gocv.Mat) ([]HandLandmarks, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hands, nil
}

func (m *MockDetector) Close() error {
	return nil
}

// ThumbsUpLandmarks returns landmarks for a thumbs up gesture
func ThumbsUpLandmarks() HandLandmarks {
	return HandLandmarks{
		Points: [NumLandmarks]Point3D{
			Wrist:     {0.5, 0.8, 0},
			ThumbCMC:  {0.45, 0.7, 0},
			ThumbMCP:  {0.4, 0.6, 0},
			ThumbIP:   {0.38, 0.45, 0},
			ThumbTip:  {0.35, 0.3, 0}, // Thumb pointing up
			IndexMCP:  {0.45, 0.65, 0},
			IndexPIP:  {0.45, 0.7, 0},
			IndexDIP:  {0.45, 0.75, 0},
			IndexTip:  {0.45, 0.8, 0}, // Finger curled
			MiddleMCP: {0.5, 0.65, 0},
			MiddlePIP: {0.5, 0.7, 0},
			MiddleDIP: {0.5, 0.75, 0},
			MiddleTip: {0.5, 0.8, 0},
			RingMCP:   {0.55, 0.65, 0},
			RingPIP:   {0.55, 0.7, 0},
			RingDIP:   {0.55, 0.75, 0},
			RingTip:   {0.55, 0.8, 0},
			PinkyMCP:  {0.6, 0.65, 0},
			PinkyPIP:  {0.6, 0.7, 0},
			PinkyDIP:  {0.6, 0.75, 0},
			PinkyTip:  {0.6, 0.8, 0},
		},
		Handedness: "Right",
		Score:      0.95,
	}
}

// OpenPalmLandmarks returns landmarks for an open palm gesture
func OpenPalmLandmarks() HandLandmarks {
	return HandLandmarks{
		Points: [NumLandmarks]Point3D{
			Wrist:     {0.5, 0.9, 0},
			ThumbCMC:  {0.35, 0.8, 0},
			ThumbMCP:  {0.25, 0.7, 0},
			ThumbIP:   {0.2, 0.6, 0},
			ThumbTip:  {0.15, 0.5, 0},
			IndexMCP:  {0.4, 0.7, 0},
			IndexPIP:  {0.38, 0.5, 0},
			IndexDIP:  {0.37, 0.35, 0},
			IndexTip:  {0.36, 0.2, 0},
			MiddleMCP: {0.5, 0.68, 0},
			MiddlePIP: {0.5, 0.48, 0},
			MiddleDIP: {0.5, 0.32, 0},
			MiddleTip: {0.5, 0.15, 0},
			RingMCP:   {0.6, 0.7, 0},
			RingPIP:   {0.62, 0.5, 0},
			RingDIP:   {0.63, 0.35, 0},
			RingTip:   {0.64, 0.2, 0},
			PinkyMCP:  {0.7, 0.72, 0},
			PinkyPIP:  {0.73, 0.55, 0},
			PinkyDIP:  {0.75, 0.42, 0},
			PinkyTip:  {0.77, 0.3, 0},
		},
		Handedness: "Right",
		Score:      0.92,
	}
}
```

**Step 2: Write test**

```go
// internal/detector/detector_test.go
package detector

import (
	"testing"
)

func TestHandLandmarks_Normalize(t *testing.T) {
	landmarks := ThumbsUpLandmarks()
	normalized := landmarks.Normalize()

	// Wrist should be at origin after normalization
	if normalized.Points[Wrist].X != 0 || normalized.Points[Wrist].Y != 0 {
		t.Errorf("wrist not at origin: got (%f, %f)",
			normalized.Points[Wrist].X, normalized.Points[Wrist].Y)
	}

	// Handedness should be preserved
	if normalized.Handedness != landmarks.Handedness {
		t.Errorf("handedness changed: got %s, want %s",
			normalized.Handedness, landmarks.Handedness)
	}
}

func TestMockDetector(t *testing.T) {
	mock := NewMockDetector()

	// No hands initially
	hands, err := mock.Detect(nil)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(hands) != 0 {
		t.Errorf("expected 0 hands, got %d", len(hands))
	}

	// Set hands
	mock.SetHands([]HandLandmarks{ThumbsUpLandmarks()})
	hands, err = mock.Detect(nil)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(hands) != 1 {
		t.Errorf("expected 1 hand, got %d", len(hands))
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/detector/... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/detector/mock.go internal/detector/detector_test.go
git commit -m "feat: add mock hand detector for testing"
```

---

## Phase 4: Gesture Matching

### Task 8: Static Gesture Matcher

**Files:**
- Create: `internal/gesture/matcher.go`
- Create: `internal/gesture/matcher_test.go`

**Step 1: Write the failing test**

```go
// internal/gesture/matcher_test.go
package gesture

import (
	"testing"

	"github.com/ayusman/kuchipudi/internal/detector"
)

func TestStaticMatcher_Match(t *testing.T) {
	// Create a template from thumbs up
	template := &Template{
		ID:        "thumbs-up",
		Name:      "Thumbs Up",
		Type:      TypeStatic,
		Landmarks: detector.ThumbsUpLandmarks().Normalize().Points[:],
		Tolerance: 0.3,
	}

	matcher := NewStaticMatcher()
	matcher.AddTemplate(template)

	// Test with same gesture - should match
	input := detector.ThumbsUpLandmarks()
	matches := matcher.Match(&input)

	if len(matches) == 0 {
		t.Fatal("expected match for identical gesture")
	}
	if matches[0].Template.ID != "thumbs-up" {
		t.Errorf("wrong template matched: %s", matches[0].Template.ID)
	}

	// Test with different gesture - should not match
	input2 := detector.OpenPalmLandmarks()
	matches2 := matcher.Match(&input2)

	for _, m := range matches2 {
		if m.Template.ID == "thumbs-up" && m.Score > 0.7 {
			t.Errorf("thumbs-up should not match open palm with high score: %f", m.Score)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/gesture/... -v`
Expected: FAIL - package not found

**Step 3: Write matcher implementation**

```go
// internal/gesture/matcher.go
package gesture

import (
	"math"
	"sort"

	"github.com/ayusman/kuchipudi/internal/detector"
)

type Type string

const (
	TypeStatic  Type = "static"
	TypeDynamic Type = "dynamic"
)

type Template struct {
	ID        string
	Name      string
	Type      Type
	Landmarks []detector.Point3D // For static gestures
	Path      []PathPoint         // For dynamic gestures
	Tolerance float64
}

type PathPoint struct {
	X         float64
	Y         float64
	Timestamp int64 // milliseconds
}

type Match struct {
	Template *Template
	Score    float64 // 0-1, higher is better
	Distance float64 // Raw distance value
}

type StaticMatcher struct {
	templates []*Template
}

func NewStaticMatcher() *StaticMatcher {
	return &StaticMatcher{
		templates: make([]*Template, 0),
	}
}

func (m *StaticMatcher) AddTemplate(t *Template) {
	m.templates = append(m.templates, t)
}

func (m *StaticMatcher) RemoveTemplate(id string) {
	for i, t := range m.templates {
		if t.ID == id {
			m.templates = append(m.templates[:i], m.templates[i+1:]...)
			return
		}
	}
}

func (m *StaticMatcher) Match(hand *detector.HandLandmarks) []Match {
	normalized := hand.Normalize()
	var matches []Match

	for _, template := range m.templates {
		if template.Type != TypeStatic {
			continue
		}

		dist := euclideanDistance(normalized.Points[:], template.Landmarks)
		score := 1.0 / (1.0 + dist) // Convert distance to 0-1 score

		if dist <= template.Tolerance {
			matches = append(matches, Match{
				Template: template,
				Score:    score,
				Distance: dist,
			})
		}
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

func euclideanDistance(a, b []detector.Point3D) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	var sum float64
	for i := range a {
		dx := a[i].X - b[i].X
		dy := a[i].Y - b[i].Y
		dz := a[i].Z - b[i].Z
		sum += dx*dx + dy*dy + dz*dz
	}

	return math.Sqrt(sum / float64(len(a)))
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/gesture/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/gesture/
git commit -m "feat: add static gesture matcher with Euclidean distance"
```

---

### Task 9: Dynamic Gesture Matcher (DTW)

**Files:**
- Modify: `internal/gesture/matcher.go`
- Create: `internal/gesture/dtw.go`
- Create: `internal/gesture/dtw_test.go`

**Step 1: Write the failing test**

```go
// internal/gesture/dtw_test.go
package gesture

import (
	"testing"
)

func TestDTW_IdenticalPaths(t *testing.T) {
	path := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 0.5, Y: 0.5, Timestamp: 100},
		{X: 1, Y: 1, Timestamp: 200},
	}

	dist := DTWDistance(path, path)
	if dist != 0 {
		t.Errorf("DTW of identical paths should be 0, got %f", dist)
	}
}

func TestDTW_DifferentPaths(t *testing.T) {
	path1 := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 0, Timestamp: 100}, // Moving right
	}
	path2 := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 0, Y: 1, Timestamp: 100}, // Moving up
	}

	dist := DTWDistance(path1, path2)
	if dist <= 0 {
		t.Errorf("DTW of different paths should be > 0, got %f", dist)
	}
}

func TestDTW_SpeedInvariant(t *testing.T) {
	// Same path, different speeds
	fast := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 1, Y: 1, Timestamp: 100},
	}
	slow := []PathPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 0.5, Y: 0.5, Timestamp: 100},
		{X: 1, Y: 1, Timestamp: 200},
	}

	dist := DTWDistance(fast, slow)
	// Should be small since the paths are the same trajectory
	if dist > 0.5 {
		t.Errorf("DTW should handle speed differences, got %f", dist)
	}
}

func TestDynamicMatcher_Match(t *testing.T) {
	swipeLeft := &Template{
		ID:        "swipe-left",
		Name:      "Swipe Left",
		Type:      TypeDynamic,
		Path: []PathPoint{
			{X: 0.8, Y: 0.5, Timestamp: 0},
			{X: 0.5, Y: 0.5, Timestamp: 100},
			{X: 0.2, Y: 0.5, Timestamp: 200},
		},
		Tolerance: 0.5,
	}

	matcher := NewDynamicMatcher()
	matcher.AddTemplate(swipeLeft)

	// Test with similar swipe
	input := []PathPoint{
		{X: 0.9, Y: 0.5, Timestamp: 0},
		{X: 0.6, Y: 0.5, Timestamp: 150},
		{X: 0.3, Y: 0.5, Timestamp: 300},
	}

	matches := matcher.Match(input)
	if len(matches) == 0 {
		t.Fatal("expected match for swipe left")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/gesture/... -v -run DTW`
Expected: FAIL - DTWDistance not defined

**Step 3: Write DTW implementation**

```go
// internal/gesture/dtw.go
package gesture

import (
	"math"
)

// DTWDistance calculates Dynamic Time Warping distance between two paths
// This allows matching gestures performed at different speeds
func DTWDistance(path1, path2 []PathPoint) float64 {
	n := len(path1)
	m := len(path2)

	if n == 0 || m == 0 {
		return math.MaxFloat64
	}

	// Create cost matrix
	dtw := make([][]float64, n+1)
	for i := range dtw {
		dtw[i] = make([]float64, m+1)
		for j := range dtw[i] {
			dtw[i][j] = math.MaxFloat64
		}
	}
	dtw[0][0] = 0

	// Fill the matrix
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := pointDistance(path1[i-1], path2[j-1])
			dtw[i][j] = cost + min3(
				dtw[i-1][j],   // insertion
				dtw[i][j-1],   // deletion
				dtw[i-1][j-1], // match
			)
		}
	}

	// Normalize by path length
	return dtw[n][m] / float64(max(n, m))
}

func pointDistance(a, b PathPoint) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func min3(a, b, c float64) float64 {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DynamicMatcher matches dynamic (movement-based) gestures
type DynamicMatcher struct {
	templates []*Template
}

func NewDynamicMatcher() *DynamicMatcher {
	return &DynamicMatcher{
		templates: make([]*Template, 0),
	}
}

func (m *DynamicMatcher) AddTemplate(t *Template) {
	m.templates = append(m.templates, t)
}

func (m *DynamicMatcher) RemoveTemplate(id string) {
	for i, t := range m.templates {
		if t.ID == id {
			m.templates = append(m.templates[:i], m.templates[i+1:]...)
			return
		}
	}
}

func (m *DynamicMatcher) Match(path []PathPoint) []Match {
	if len(path) < 2 {
		return nil
	}

	// Normalize path
	normalized := normalizePath(path)
	var matches []Match

	for _, template := range m.templates {
		if template.Type != TypeDynamic {
			continue
		}

		dist := DTWDistance(normalized, template.Path)
		score := 1.0 / (1.0 + dist)

		if dist <= template.Tolerance {
			matches = append(matches, Match{
				Template: template,
				Score:    score,
				Distance: dist,
			})
		}
	}

	return matches
}

// normalizePath scales path to 0-1 range
func normalizePath(path []PathPoint) []PathPoint {
	if len(path) == 0 {
		return path
	}

	// Find bounding box
	minX, maxX := path[0].X, path[0].X
	minY, maxY := path[0].Y, path[0].Y

	for _, p := range path {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Calculate scale
	rangeX := maxX - minX
	rangeY := maxY - minY
	scale := rangeX
	if rangeY > scale {
		scale = rangeY
	}
	if scale < 0.001 {
		scale = 1
	}

	// Normalize
	normalized := make([]PathPoint, len(path))
	for i, p := range path {
		normalized[i] = PathPoint{
			X:         (p.X - minX) / scale,
			Y:         (p.Y - minY) / scale,
			Timestamp: p.Timestamp,
		}
	}

	return normalized
}
```

**Step 4: Run tests**

Run: `go test ./internal/gesture/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/gesture/dtw.go internal/gesture/dtw_test.go
git commit -m "feat: add Dynamic Time Warping for movement gesture matching"
```

---

## Phase 5: Plugin System

### Task 10: Plugin Manager

**Files:**
- Create: `internal/plugin/types.go`
- Create: `internal/plugin/manager.go`
- Create: `internal/plugin/manager_test.go`

**Step 1: Write plugin types**

```go
// internal/plugin/types.go
package plugin

import "encoding/json"

// Manifest describes a plugin
type Manifest struct {
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	Executable   string          `json:"executable"`
	Actions      []string        `json:"actions"`
	ConfigSchema json.RawMessage `json:"configSchema,omitempty"`
}

// Request is sent to plugins
type Request struct {
	Action  string          `json:"action"`  // "execute", "configure", "validate", "list-actions"
	Gesture string          `json:"gesture"` // Which gesture triggered this
	Config  json.RawMessage `json:"config"`  // Plugin-specific config
	Params  json.RawMessage `json:"params"`  // Action-specific params
}

// Response from plugins
type Response struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Plugin represents a loaded plugin
type Plugin struct {
	Manifest   Manifest
	Path       string // Path to plugin directory
	Executable string // Full path to executable
}
```

**Step 2: Write the failing test**

```go
// internal/plugin/manager_test.go
package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_Discover(t *testing.T) {
	// Create temp plugin directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	os.MkdirAll(pluginDir, 0755)

	// Write manifest
	manifest := `{
		"name": "test-plugin",
		"version": "1.0.0",
		"description": "Test plugin",
		"executable": "test-plugin",
		"actions": ["action1", "action2"]
	}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0644)

	// Create dummy executable
	os.WriteFile(filepath.Join(pluginDir, "test-plugin"), []byte("#!/bin/sh\necho ok"), 0755)

	// Discover plugins
	mgr := NewManager(tmpDir)
	err := mgr.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	plugins := mgr.List()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	if plugins[0].Manifest.Name != "test-plugin" {
		t.Errorf("plugin name = %s, want test-plugin", plugins[0].Manifest.Name)
	}
}

func TestManager_Get(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.Get("nonexistent")
	if err != ErrPluginNotFound {
		t.Errorf("Get() error = %v, want ErrPluginNotFound", err)
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/plugin/... -v`
Expected: FAIL - Manager not defined

**Step 4: Write manager implementation**

```go
// internal/plugin/manager.go
package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var ErrPluginNotFound = errors.New("plugin not found")

type Manager struct {
	pluginDir string
	plugins   map[string]*Plugin
	mu        sync.RWMutex
}

func NewManager(pluginDir string) *Manager {
	return &Manager{
		pluginDir: pluginDir,
		plugins:   make(map[string]*Plugin),
	}
}

func (m *Manager) Discover() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing plugins
	m.plugins = make(map[string]*Plugin)

	// Read plugin directory
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No plugins directory yet
		}
		return fmt.Errorf("read plugin dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(m.pluginDir, entry.Name())
		manifestPath := filepath.Join(pluginPath, "plugin.json")

		// Read manifest
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // Skip plugins without manifest
		}

		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue // Skip invalid manifests
		}

		// Verify executable exists
		execPath := filepath.Join(pluginPath, manifest.Executable)
		if _, err := os.Stat(execPath); err != nil {
			continue // Skip plugins without executable
		}

		m.plugins[manifest.Name] = &Plugin{
			Manifest:   manifest,
			Path:       pluginPath,
			Executable: execPath,
		}
	}

	return nil
}

func (m *Manager) Get(name string) (*Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, ok := m.plugins[name]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return plugin, nil
}

func (m *Manager) List() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (m *Manager) PluginDir() string {
	return m.pluginDir
}
```

**Step 5: Run tests**

Run: `go test ./internal/plugin/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/plugin/
git commit -m "feat: add plugin manager with discovery"
```

---

### Task 11: Plugin Executor

**Files:**
- Create: `internal/plugin/executor.go`
- Create: `internal/plugin/executor_test.go`

**Step 1: Write the failing test**

```go
// internal/plugin/executor_test.go
package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecutor_Execute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	// Create a simple test plugin
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "echo-plugin")
	os.MkdirAll(pluginDir, 0755)

	// Create executable that echoes success
	script := `#!/bin/sh
read input
echo '{"success": true, "data": "executed"}'
`
	execPath := filepath.Join(pluginDir, "echo-plugin")
	os.WriteFile(execPath, []byte(script), 0755)

	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "echo-plugin",
			Executable: "echo-plugin",
		},
		Path:       pluginDir,
		Executable: execPath,
	}

	executor := NewExecutor(5000) // 5s timeout
	req := &Request{
		Action:  "execute",
		Gesture: "test-gesture",
		Config:  json.RawMessage(`{}`),
	}

	resp, err := executor.Execute(plugin, req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !resp.Success {
		t.Errorf("Execute() success = false, error = %s", resp.Error)
	}
}

func TestExecutor_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "slow-plugin")
	os.MkdirAll(pluginDir, 0755)

	// Create executable that sleeps forever
	script := `#!/bin/sh
sleep 10
`
	execPath := filepath.Join(pluginDir, "slow-plugin")
	os.WriteFile(execPath, []byte(script), 0755)

	plugin := &Plugin{
		Manifest: Manifest{
			Name:       "slow-plugin",
			Executable: "slow-plugin",
		},
		Path:       pluginDir,
		Executable: execPath,
	}

	executor := NewExecutor(100) // 100ms timeout
	req := &Request{Action: "execute"}

	_, err := executor.Execute(plugin, req)
	if err == nil {
		t.Error("Execute() should timeout")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/plugin/... -v -run Executor`
Expected: FAIL - Executor not defined

**Step 3: Write executor implementation**

```go
// internal/plugin/executor.go
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type Executor struct {
	timeoutMs int
}

func NewExecutor(timeoutMs int) *Executor {
	return &Executor{timeoutMs: timeoutMs}
}

func (e *Executor) Execute(plugin *Plugin, req *Request) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, plugin.Executable)
	cmd.Dir = plugin.Path

	// Prepare input
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("plugin timeout after %dms", e.timeoutMs)
	}
	if err != nil {
		return nil, fmt.Errorf("plugin error: %w, stderr: %s", err, stderr.String())
	}

	var resp Response
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w, output: %s", err, stdout.String())
	}

	return &resp, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/plugin/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/plugin/executor.go internal/plugin/executor_test.go
git commit -m "feat: add plugin executor with timeout support"
```

---

## Phase 6: Web Server & API

### Task 12: HTTP Server Setup

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`

**Step 1: Write the failing test**

```go
// internal/server/server_test.go
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_Health(t *testing.T) {
	srv := New(Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestServer_NotFound(t *testing.T) {
	srv := New(Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server/... -v`
Expected: FAIL - package not found

**Step 3: Write server implementation**

```go
// internal/server/server.go
package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type Config struct {
	StaticDir string
}

type Server struct {
	config Config
	mux    *http.ServeMux
	start  time.Time
}

func New(config Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
		start:  time.Now(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/health", s.handleHealth)

	// Static files (if configured)
	if s.config.StaticDir != "" {
		fs := http.FileServer(http.Dir(s.config.StaticDir))
		s.mux.Handle("/", fs)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]any{
		"status": "ok",
		"uptime": time.Since(s.start).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}
```

**Step 4: Run tests**

Run: `go test ./internal/server/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/server/
git commit -m "feat: add HTTP server with health endpoint"
```

---

### Task 13: Gesture API Endpoints

**Files:**
- Create: `internal/server/api/gestures.go`
- Create: `internal/server/api/gestures_test.go`
- Modify: `internal/server/server.go`

**Step 1: Write the failing test**

```go
// internal/server/api/gestures_test.go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ayusman/kuchipudi/internal/store"
)

func TestGestureHandler_List(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	// Add test gesture
	s.Gestures().Create(&store.Gesture{
		ID:   "g1",
		Name: "test-gesture",
		Type: store.GestureTypeStatic,
	})

	handler := NewGestureHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/api/gestures", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Gestures []store.Gesture `json:"gestures"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(resp.Gestures) != 1 {
		t.Errorf("len(gestures) = %d, want 1", len(resp.Gestures))
	}
}

func TestGestureHandler_Create(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	handler := NewGestureHandler(s)

	body := `{"name": "new-gesture", "type": "static"}`
	req := httptest.NewRequest(http.MethodPost, "/api/gestures", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify created
	gestures, _ := s.Gestures().List()
	if len(gestures) != 1 {
		t.Errorf("gesture not created")
	}
}

func TestGestureHandler_Delete(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	s.Gestures().Create(&store.Gesture{
		ID:   "g1",
		Name: "to-delete",
		Type: store.GestureTypeStatic,
	})

	handler := NewGestureHandler(s)

	req := httptest.NewRequest(http.MethodDelete, "/api/gestures/g1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify deleted
	_, err := s.Gestures().GetByID("g1")
	if err != store.ErrNotFound {
		t.Errorf("gesture not deleted")
	}
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	return s
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server/api/... -v`
Expected: FAIL - package not found

**Step 3: Write gesture handler**

```go
// internal/server/api/gestures.go
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ayusman/kuchipudi/internal/store"
	"github.com/google/uuid"
)

type GestureHandler struct {
	store *store.Store
}

func NewGestureHandler(s *store.Store) *GestureHandler {
	return &GestureHandler{store: s}
}

func (h *GestureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/gestures or /api/gestures/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/gestures")
	path = strings.TrimPrefix(path, "/")

	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.list(w, r)
		} else {
			h.get(w, r, path)
		}
	case http.MethodPost:
		h.create(w, r)
	case http.MethodPut:
		if path != "" {
			h.update(w, r, path)
		} else {
			http.Error(w, "missing gesture id", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if path != "" {
			h.delete(w, r, path)
		} else {
			http.Error(w, "missing gesture id", http.StatusBadRequest)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *GestureHandler) list(w http.ResponseWriter, r *http.Request) {
	gestures, err := h.store.Gestures().List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"gestures": gestures})
}

func (h *GestureHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	gesture, err := h.store.Gestures().GetByID(id)
	if err == store.ErrNotFound {
		http.Error(w, "gesture not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gesture)
}

type createGestureRequest struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Tolerance float64 `json:"tolerance"`
}

func (h *GestureHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createGestureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if req.Type != string(store.GestureTypeStatic) && req.Type != string(store.GestureTypeDynamic) {
		http.Error(w, "type must be 'static' or 'dynamic'", http.StatusBadRequest)
		return
	}

	tolerance := req.Tolerance
	if tolerance == 0 {
		tolerance = 0.15 // default
	}

	gesture := &store.Gesture{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      store.GestureType(req.Type),
		Tolerance: tolerance,
	}

	if err := h.store.Gestures().Create(gesture); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(gesture)
}

func (h *GestureHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	var req createGestureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	gesture, err := h.store.Gestures().GetByID(id)
	if err == store.ErrNotFound {
		http.Error(w, "gesture not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Name != "" {
		gesture.Name = req.Name
	}
	if req.Tolerance != 0 {
		gesture.Tolerance = req.Tolerance
	}

	if err := h.store.Gestures().Update(gesture); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gesture)
}

func (h *GestureHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Gestures().Delete(id); err == store.ErrNotFound {
		http.Error(w, "gesture not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 4: Add uuid dependency**

Run: `go get github.com/google/uuid`
Expected: Adds uuid to go.mod

**Step 5: Run tests**

Run: `go test ./internal/server/api/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/server/api/ go.mod go.sum
git commit -m "feat: add gesture API endpoints"
```

---

## Phase 7: System Tray

### Task 14: macOS Tray App

**Files:**
- Create: `internal/tray/tray.go`
- Modify: `cmd/kuchipudi/main.go`

**Step 1: Add systray dependency**

Run: `go get github.com/getlantern/systray`
Expected: Adds systray to go.mod

**Step 2: Write tray implementation**

```go
// internal/tray/tray.go
package tray

import (
	"fmt"

	"github.com/getlantern/systray"
)

type Tray struct {
	onToggle   func(enabled bool)
	onSettings func()
	onQuit     func()
	enabled    bool
}

func New() *Tray {
	return &Tray{
		enabled: true,
	}
}

func (t *Tray) OnToggle(fn func(enabled bool)) {
	t.onToggle = fn
}

func (t *Tray) OnSettings(fn func()) {
	t.onSettings = fn
}

func (t *Tray) OnQuit(fn func()) {
	t.onQuit = fn
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetTitle("👋")
	systray.SetTooltip("Kuchipudi - Hand Gesture Recognition")

	// Menu items
	mToggle := systray.AddMenuItem("● Enabled", "Toggle gesture recognition")
	systray.AddSeparator()
	mLast := systray.AddMenuItem("Last: none", "Most recent gesture")
	mLast.Disable()
	systray.AddSeparator()
	mSettings := systray.AddMenuItem("Open Settings...", "Configure gestures and actions")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit Kuchipudi")

	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				t.enabled = !t.enabled
				if t.enabled {
					mToggle.SetTitle("● Enabled")
				} else {
					mToggle.SetTitle("○ Disabled")
				}
				if t.onToggle != nil {
					t.onToggle(t.enabled)
				}

			case <-mSettings.ClickedCh:
				if t.onSettings != nil {
					t.onSettings()
				}

			case <-mQuit.ClickedCh:
				if t.onQuit != nil {
					t.onQuit()
				}
				systray.Quit()
			}
		}
	}()
}

func (t *Tray) onExit() {
	// Cleanup if needed
}

func (t *Tray) SetLastGesture(name string) {
	// Note: systray doesn't support updating menu items after creation
	// This would need a more sophisticated approach with menu recreation
	fmt.Printf("Last gesture: %s\n", name)
}

func (t *Tray) IsEnabled() bool {
	return t.enabled
}
```

**Step 3: Update main.go**

```go
// cmd/kuchipudi/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ayusman/kuchipudi/internal/server"
	"github.com/ayusman/kuchipudi/internal/store"
	"github.com/ayusman/kuchipudi/internal/tray"
)

const (
	defaultPort = "9847"
	appName     = "kuchipudi"
)

func main() {
	// Setup data directory
	dataDir, err := getDataDir()
	if err != nil {
		log.Fatalf("failed to get data directory: %v", err)
	}

	// Initialize store
	dbPath := filepath.Join(dataDir, "data.db")
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()

	// Initialize web server
	webDir := filepath.Join(dataDir, "web")
	srv := server.New(server.Config{
		StaticDir: webDir,
	})

	// Start server in background
	addr := "127.0.0.1:" + defaultPort
	go func() {
		log.Printf("Starting server on http://%s", addr)
		if err := srv.ListenAndServe(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Setup tray
	t := tray.New()

	t.OnToggle(func(enabled bool) {
		if enabled {
			log.Println("Gesture recognition enabled")
		} else {
			log.Println("Gesture recognition disabled")
		}
	})

	t.OnSettings(func() {
		url := fmt.Sprintf("http://%s", addr)
		openBrowser(url)
	})

	t.OnQuit(func() {
		log.Println("Shutting down...")
		s.Close()
	})

	// Run tray (blocks)
	t.Run()
}

func getDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dataDir := filepath.Join(home, "."+appName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}

	return dataDir, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		log.Printf("Please open %s in your browser", url)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}
```

**Step 4: Run build to verify**

Run: `go build -o bin/kuchipudi ./cmd/kuchipudi`
Expected: Builds successfully

**Step 5: Commit**

```bash
git add internal/tray/ cmd/kuchipudi/main.go go.mod go.sum
git commit -m "feat: add macOS system tray with settings and toggle"
```

---

## Phase 8: Integration & Main Loop

### Task 15: Detection Pipeline

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/pipeline.go`

**Step 1: Write app configuration**

```go
// internal/app/app.go
package app

import (
	"log"
	"sync"

	"github.com/ayusman/kuchipudi/internal/capture"
	"github.com/ayusman/kuchipudi/internal/detector"
	"github.com/ayusman/kuchipudi/internal/gesture"
	"github.com/ayusman/kuchipudi/internal/plugin"
	"github.com/ayusman/kuchipudi/internal/store"
)

type Config struct {
	Store        *store.Store
	PluginDir    string
	CameraID     int
	MotionThresh float64
}

type App struct {
	config        Config
	camera        *capture.Camera
	motion        *capture.MotionDetector
	detector      detector.Detector
	staticMatcher *gesture.StaticMatcher
	dynamicMatcher *gesture.DynamicMatcher
	pluginMgr     *plugin.Manager
	pluginExec    *plugin.Executor

	enabled bool
	mu      sync.RWMutex
	stopCh  chan struct{}
}

func New(config Config) *App {
	return &App{
		config:         config,
		camera:         capture.NewCamera(config.CameraID),
		motion:         capture.NewMotionDetector(config.MotionThresh),
		detector:       detector.NewMockDetector(), // Replace with real MediaPipe later
		staticMatcher:  gesture.NewStaticMatcher(),
		dynamicMatcher: gesture.NewDynamicMatcher(),
		pluginMgr:      plugin.NewManager(config.PluginDir),
		pluginExec:     plugin.NewExecutor(5000),
		enabled:        true,
		stopCh:         make(chan struct{}),
	}
}

func (a *App) SetEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = enabled
	log.Printf("App enabled: %v", enabled)
}

func (a *App) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

func (a *App) LoadGestures() error {
	gestures, err := a.config.Store.Gestures().List()
	if err != nil {
		return err
	}

	for _, g := range gestures {
		template := &gesture.Template{
			ID:        g.ID,
			Name:      g.Name,
			Tolerance: g.Tolerance,
		}

		if g.Type == store.GestureTypeStatic {
			template.Type = gesture.TypeStatic
			// TODO: Load landmarks from DB
			a.staticMatcher.AddTemplate(template)
		} else {
			template.Type = gesture.TypeDynamic
			// TODO: Load path from DB
			a.dynamicMatcher.AddTemplate(template)
		}
	}

	log.Printf("Loaded %d gestures", len(gestures))
	return nil
}

func (a *App) Start() error {
	if err := a.camera.Open(); err != nil {
		return err
	}

	if err := a.pluginMgr.Discover(); err != nil {
		log.Printf("Warning: plugin discovery failed: %v", err)
	}

	go a.runPipeline()

	log.Println("App started")
	return nil
}

func (a *App) Stop() {
	close(a.stopCh)
	a.camera.Close()
	a.motion.Close()
	log.Println("App stopped")
}
```

**Step 2: Write detection pipeline**

```go
// internal/app/pipeline.go
package app

import (
	"log"
	"time"

	"github.com/ayusman/kuchipudi/internal/gesture"
)

func (a *App) runPipeline() {
	idleFPS := 5
	activeFPS := 15
	currentFPS := idleFPS
	lastMotion := time.Now()
	inactiveTimeout := 2 * time.Second

	// Path buffer for dynamic gestures
	var pathBuffer []gesture.PathPoint

	ticker := time.NewTicker(time.Second / time.Duration(currentFPS))
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			if !a.IsEnabled() {
				continue
			}

			frame, err := a.camera.ReadFrame()
			if err != nil {
				continue
			}

			// Motion detection
			motionDetected, _ := a.motion.Detect(frame)

			if motionDetected {
				lastMotion = time.Now()

				// Switch to active mode
				if currentFPS != activeFPS {
					currentFPS = activeFPS
					ticker.Reset(time.Second / time.Duration(currentFPS))
					log.Println("Switching to active mode")
				}

				// Run hand detection
				hands, err := a.detector.Detect(frame)
				if err != nil {
					frame.Close()
					continue
				}

				for _, hand := range hands {
					// Check static gestures
					staticMatches := a.staticMatcher.Match(&hand)
					if len(staticMatches) > 0 {
						best := staticMatches[0]
						log.Printf("Static gesture matched: %s (score: %.2f)", best.Template.Name, best.Score)
						a.executeAction(best.Template.ID, best.Template.Name)
					}

					// Track path for dynamic gestures
					pathBuffer = append(pathBuffer, gesture.PathPoint{
						X:         hand.Points[0].X, // Use wrist position
						Y:         hand.Points[0].Y,
						Timestamp: time.Now().UnixMilli(),
					})

					// Keep last 60 frames (~4 seconds at 15 FPS)
					if len(pathBuffer) > 60 {
						pathBuffer = pathBuffer[1:]
					}

					// Check dynamic gestures
					if len(pathBuffer) > 10 {
						dynamicMatches := a.dynamicMatcher.Match(pathBuffer)
						if len(dynamicMatches) > 0 {
							best := dynamicMatches[0]
							log.Printf("Dynamic gesture matched: %s (score: %.2f)", best.Template.Name, best.Score)
							a.executeAction(best.Template.ID, best.Template.Name)
							// Clear buffer after match to prevent repeated triggers
							pathBuffer = pathBuffer[:0]
						}
					}
				}
			} else if time.Since(lastMotion) > inactiveTimeout && currentFPS != idleFPS {
				// Switch back to idle mode
				currentFPS = idleFPS
				ticker.Reset(time.Second / time.Duration(currentFPS))
				pathBuffer = pathBuffer[:0]
				log.Println("Switching to idle mode")
			}

			frame.Close()
		}
	}
}

func (a *App) executeAction(gestureID, gestureName string) {
	// TODO: Look up action mapping from store
	// For now, just log
	log.Printf("Execute action for gesture: %s (%s)", gestureName, gestureID)
}
```

**Step 3: Run build to verify**

Run: `go build -o bin/kuchipudi ./cmd/kuchipudi`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add internal/app/
git commit -m "feat: add detection pipeline with idle/active modes"
```

---

## Phase 9: Bundled Plugins

### Task 16: System Control Plugin

**Files:**
- Create: `plugins/system-control/main.go`
- Create: `plugins/system-control/plugin.json`

**Step 1: Write plugin manifest**

```json
{
    "name": "system-control",
    "version": "1.0.0",
    "description": "Control system volume, brightness, and media playback",
    "executable": "system-control",
    "actions": [
        "volume-up",
        "volume-down",
        "volume-mute",
        "brightness-up",
        "brightness-down",
        "media-play-pause",
        "media-next",
        "media-prev"
    ]
}
```

**Step 2: Write plugin**

```go
// plugins/system-control/main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type Request struct {
	Action  string          `json:"action"`
	Gesture string          `json:"gesture"`
	Config  json.RawMessage `json:"config"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func main() {
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		respond(false, fmt.Sprintf("invalid request: %v", err))
		return
	}

	var params struct {
		ActionName string `json:"action_name"`
	}
	json.Unmarshal(req.Params, &params)

	switch req.Action {
	case "execute":
		executeAction(params.ActionName)
	default:
		respond(false, fmt.Sprintf("unknown action: %s", req.Action))
	}
}

func executeAction(action string) {
	var script string

	switch action {
	case "volume-up":
		script = `osascript -e "set volume output volume ((output volume of (get volume settings)) + 10)"`
	case "volume-down":
		script = `osascript -e "set volume output volume ((output volume of (get volume settings)) - 10)"`
	case "volume-mute":
		script = `osascript -e "set volume output muted (not (output muted of (get volume settings)))"`
	case "brightness-up":
		// Requires brightness control utility
		script = `brightness +0.1 2>/dev/null || echo "brightness utility not installed"`
	case "brightness-down":
		script = `brightness -0.1 2>/dev/null || echo "brightness utility not installed"`
	case "media-play-pause":
		script = `osascript -e 'tell application "System Events" to key code 16 using {command down, option down}'`
	case "media-next":
		script = `osascript -e 'tell application "System Events" to key code 17 using {command down, option down}'`
	case "media-prev":
		script = `osascript -e 'tell application "System Events" to key code 18 using {command down, option down}'`
	default:
		respond(false, fmt.Sprintf("unknown action: %s", action))
		return
	}

	cmd := exec.Command("sh", "-c", script)
	if err := cmd.Run(); err != nil {
		respond(false, fmt.Sprintf("execution failed: %v", err))
		return
	}

	respond(true, "")
}

func respond(success bool, errMsg string) {
	resp := Response{Success: success, Error: errMsg}
	json.NewEncoder(os.Stdout).Encode(resp)
}
```

**Step 3: Build plugin**

Run: `cd plugins/system-control && go build -o system-control . && cd ../..`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add plugins/system-control/
git commit -m "feat: add system-control plugin for volume/media"
```

---

### Task 17: Keyboard Plugin

**Files:**
- Create: `plugins/keyboard/main.go`
- Create: `plugins/keyboard/plugin.json`

**Step 1: Write plugin manifest**

```json
{
    "name": "keyboard",
    "version": "1.0.0",
    "description": "Send keyboard shortcuts and keystrokes",
    "executable": "keyboard",
    "actions": [
        "keystroke",
        "shortcut"
    ],
    "configSchema": {
        "keystroke": {
            "key": "string",
            "modifiers": ["command", "option", "control", "shift"]
        }
    }
}
```

**Step 2: Write plugin**

```go
// plugins/keyboard/main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Request struct {
	Action  string          `json:"action"`
	Gesture string          `json:"gesture"`
	Config  json.RawMessage `json:"config"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type KeystrokeParams struct {
	Key       string   `json:"key"`
	Modifiers []string `json:"modifiers"`
}

func main() {
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		respond(false, fmt.Sprintf("invalid request: %v", err))
		return
	}

	switch req.Action {
	case "execute":
		var params KeystrokeParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			respond(false, fmt.Sprintf("invalid params: %v", err))
			return
		}
		executeKeystroke(params)
	default:
		respond(false, fmt.Sprintf("unknown action: %s", req.Action))
	}
}

func executeKeystroke(params KeystrokeParams) {
	if params.Key == "" {
		respond(false, "key required")
		return
	}

	// Build AppleScript
	var modParts []string
	for _, mod := range params.Modifiers {
		switch strings.ToLower(mod) {
		case "command", "cmd":
			modParts = append(modParts, "command down")
		case "option", "alt":
			modParts = append(modParts, "option down")
		case "control", "ctrl":
			modParts = append(modParts, "control down")
		case "shift":
			modParts = append(modParts, "shift down")
		}
	}

	var script string
	if len(modParts) > 0 {
		script = fmt.Sprintf(
			`osascript -e 'tell application "System Events" to keystroke "%s" using {%s}'`,
			params.Key,
			strings.Join(modParts, ", "),
		)
	} else {
		script = fmt.Sprintf(
			`osascript -e 'tell application "System Events" to keystroke "%s"'`,
			params.Key,
		)
	}

	cmd := exec.Command("sh", "-c", script)
	if err := cmd.Run(); err != nil {
		respond(false, fmt.Sprintf("execution failed: %v", err))
		return
	}

	respond(true, "")
}

func respond(success bool, errMsg string) {
	resp := Response{Success: success, Error: errMsg}
	json.NewEncoder(os.Stdout).Encode(resp)
}
```

**Step 3: Build plugin**

Run: `cd plugins/keyboard && go build -o keyboard . && cd ../..`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add plugins/keyboard/
git commit -m "feat: add keyboard plugin for shortcuts"
```

---

## Phase 10: Web UI

### Task 18: Basic Web Interface

**Files:**
- Create: `web/index.html`
- Create: `web/css/style.css`
- Create: `web/js/app.js`

**Step 1: Create HTML**

```html
<!-- web/index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kuchipudi - Hand Gesture Control</title>
    <link rel="stylesheet" href="/css/style.css">
</head>
<body>
    <header>
        <h1>👋 Kuchipudi</h1>
        <nav>
            <a href="#dashboard" class="active">Dashboard</a>
            <a href="#gestures">Gestures</a>
            <a href="#actions">Actions</a>
            <a href="#settings">Settings</a>
        </nav>
    </header>

    <main>
        <section id="dashboard" class="page active">
            <h2>Dashboard</h2>
            <div class="status-card">
                <div class="status-indicator online"></div>
                <span>Recognition Active</span>
            </div>
            <div class="stats">
                <div class="stat">
                    <span class="value" id="gesture-count">0</span>
                    <span class="label">Gestures</span>
                </div>
                <div class="stat">
                    <span class="value" id="action-count">0</span>
                    <span class="label">Actions</span>
                </div>
            </div>
            <div class="recent-gestures">
                <h3>Recent Activity</h3>
                <ul id="recent-list"></ul>
            </div>
        </section>

        <section id="gestures" class="page">
            <h2>Gestures</h2>
            <button id="add-gesture-btn" class="primary">+ Add Gesture</button>
            <div id="gesture-list" class="card-grid"></div>
        </section>

        <section id="actions" class="page">
            <h2>Actions</h2>
            <p>Map gestures to actions</p>
            <div id="action-mappings"></div>
        </section>

        <section id="settings" class="page">
            <h2>Settings</h2>
            <form id="settings-form">
                <label>
                    Camera
                    <select id="camera-select">
                        <option value="0">Default Camera</option>
                    </select>
                </label>
                <label>
                    Motion Sensitivity
                    <input type="range" id="motion-threshold" min="1" max="20" value="5">
                </label>
                <label>
                    <input type="checkbox" id="start-at-login"> Start at login
                </label>
                <button type="submit" class="primary">Save Settings</button>
            </form>
        </section>
    </main>

    <script src="/js/app.js"></script>
</body>
</html>
```

**Step 2: Create CSS**

```css
/* web/css/style.css */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

:root {
    --bg: #1a1a2e;
    --bg-light: #16213e;
    --primary: #e94560;
    --text: #eee;
    --text-muted: #888;
    --border: #333;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: var(--bg);
    color: var(--text);
    min-height: 100vh;
}

header {
    background: var(--bg-light);
    padding: 1rem 2rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
}

header h1 {
    font-size: 1.5rem;
}

nav a {
    color: var(--text-muted);
    text-decoration: none;
    margin-left: 2rem;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    transition: all 0.2s;
}

nav a:hover, nav a.active {
    color: var(--text);
    background: var(--bg);
}

main {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
}

.page {
    display: none;
}

.page.active {
    display: block;
}

h2 {
    margin-bottom: 1.5rem;
}

.status-card {
    background: var(--bg-light);
    padding: 1rem 1.5rem;
    border-radius: 8px;
    display: inline-flex;
    align-items: center;
    gap: 0.75rem;
}

.status-indicator {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: var(--text-muted);
}

.status-indicator.online {
    background: #4caf50;
    box-shadow: 0 0 8px #4caf50;
}

.stats {
    display: flex;
    gap: 2rem;
    margin: 2rem 0;
}

.stat {
    background: var(--bg-light);
    padding: 1.5rem 2rem;
    border-radius: 8px;
    text-align: center;
}

.stat .value {
    display: block;
    font-size: 2.5rem;
    font-weight: bold;
    color: var(--primary);
}

.stat .label {
    color: var(--text-muted);
}

.card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
    gap: 1rem;
    margin-top: 1rem;
}

.card {
    background: var(--bg-light);
    padding: 1.5rem;
    border-radius: 8px;
    border: 1px solid var(--border);
}

.card h3 {
    margin-bottom: 0.5rem;
}

.card .type {
    color: var(--text-muted);
    font-size: 0.875rem;
}

button {
    background: var(--bg-light);
    color: var(--text);
    border: 1px solid var(--border);
    padding: 0.75rem 1.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 1rem;
    transition: all 0.2s;
}

button:hover {
    border-color: var(--primary);
}

button.primary {
    background: var(--primary);
    border-color: var(--primary);
}

button.primary:hover {
    opacity: 0.9;
}

label {
    display: block;
    margin-bottom: 1rem;
}

input[type="range"] {
    width: 100%;
    margin-top: 0.5rem;
}

select {
    width: 100%;
    padding: 0.5rem;
    margin-top: 0.5rem;
    background: var(--bg);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 4px;
}

.recent-gestures {
    margin-top: 2rem;
}

.recent-gestures h3 {
    margin-bottom: 1rem;
}

.recent-gestures ul {
    list-style: none;
}

.recent-gestures li {
    padding: 0.75rem;
    border-bottom: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
}
```

**Step 3: Create JavaScript**

```javascript
// web/js/app.js
const API_BASE = '/api';

// Navigation
document.querySelectorAll('nav a').forEach(link => {
    link.addEventListener('click', (e) => {
        e.preventDefault();
        const target = link.getAttribute('href').slice(1);

        document.querySelectorAll('nav a').forEach(l => l.classList.remove('active'));
        document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));

        link.classList.add('active');
        document.getElementById(target).classList.add('active');
    });
});

// API helpers
async function fetchJSON(url, options = {}) {
    const res = await fetch(API_BASE + url, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options.headers,
        },
    });
    return res.json();
}

// Load gestures
async function loadGestures() {
    try {
        const data = await fetchJSON('/gestures');
        const gestures = data.gestures || [];

        document.getElementById('gesture-count').textContent = gestures.length;

        const list = document.getElementById('gesture-list');
        list.innerHTML = gestures.map(g => `
            <div class="card">
                <h3>${g.name}</h3>
                <span class="type">${g.type}</span>
            </div>
        `).join('');
    } catch (err) {
        console.error('Failed to load gestures:', err);
    }
}

// Health check
async function checkHealth() {
    try {
        const data = await fetchJSON('/health');
        if (data.status === 'ok') {
            document.querySelector('.status-indicator').classList.add('online');
        }
    } catch (err) {
        document.querySelector('.status-indicator').classList.remove('online');
    }
}

// Initialize
loadGestures();
checkHealth();
setInterval(checkHealth, 5000);
```

**Step 4: Commit**

```bash
git add web/
git commit -m "feat: add basic web UI for configuration"
```

---

## Phase 11: Wiring & Polish

### Task 19: Wire Server to Store and API

**Files:**
- Modify: `internal/server/server.go`
- Modify: `cmd/kuchipudi/main.go`

**Step 1: Update server to accept store and register API handlers**

```go
// internal/server/server.go
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ayusman/kuchipudi/internal/server/api"
	"github.com/ayusman/kuchipudi/internal/store"
)

type Config struct {
	StaticDir string
	Store     *store.Store
}

type Server struct {
	config Config
	mux    *http.ServeMux
	start  time.Time
}

func New(config Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
		start:  time.Now(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/health", s.handleHealth)

	// Gesture API
	if s.config.Store != nil {
		gestureHandler := api.NewGestureHandler(s.config.Store)
		s.mux.Handle("/api/gestures", gestureHandler)
		s.mux.Handle("/api/gestures/", gestureHandler)
	}

	// Static files
	if s.config.StaticDir != "" {
		fs := http.FileServer(http.Dir(s.config.StaticDir))
		s.mux.Handle("/", fs)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]any{
		"status": "ok",
		"uptime": time.Since(s.start).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}
```

**Step 2: Update main.go to pass store and use correct web directory**

```go
// cmd/kuchipudi/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ayusman/kuchipudi/internal/server"
	"github.com/ayusman/kuchipudi/internal/store"
	"github.com/ayusman/kuchipudi/internal/tray"
)

const (
	defaultPort = "9847"
	appName     = "kuchipudi"
)

func main() {
	// Setup data directory
	dataDir, err := getDataDir()
	if err != nil {
		log.Fatalf("failed to get data directory: %v", err)
	}

	// Initialize store
	dbPath := filepath.Join(dataDir, "data.db")
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()

	// Find web directory (in dev, it's in project root; in production, it's in data dir)
	webDir := findWebDir(dataDir)

	// Initialize web server with store
	srv := server.New(server.Config{
		StaticDir: webDir,
		Store:     s,
	})

	// Start server in background
	addr := "127.0.0.1:" + defaultPort
	go func() {
		log.Printf("Starting server on http://%s", addr)
		if err := srv.ListenAndServe(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Setup tray
	t := tray.New()

	t.OnToggle(func(enabled bool) {
		if enabled {
			log.Println("Gesture recognition enabled")
		} else {
			log.Println("Gesture recognition disabled")
		}
	})

	t.OnSettings(func() {
		url := fmt.Sprintf("http://%s", addr)
		openBrowser(url)
	})

	t.OnQuit(func() {
		log.Println("Shutting down...")
		s.Close()
	})

	// Run tray (blocks)
	t.Run()
}

func getDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dataDir := filepath.Join(home, "."+appName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}

	return dataDir, nil
}

func findWebDir(dataDir string) string {
	// Check if running in development (web dir in current directory or parent)
	candidates := []string{
		"web",
		"../web",
		"../../web",
		filepath.Join(dataDir, "web"),
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(dir)
			log.Printf("Using web directory: %s", absPath)
			return absPath
		}
	}

	log.Println("Warning: web directory not found")
	return ""
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		log.Printf("Please open %s in your browser", url)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}
```

**Step 3: Run build and verify**

Run: `go build -o bin/kuchipudi ./cmd/kuchipudi`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add internal/server/server.go cmd/kuchipudi/main.go
git commit -m "feat: wire server to store and serve web UI"
```

---

### Task 20: Makefile

**Files:**
- Create: `Makefile`

**Step 1: Write Makefile**

```makefile
# Makefile
.PHONY: build run test clean install plugins

APP_NAME := kuchipudi
BIN_DIR := bin
PLUGIN_DIR := plugins

build:
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

run: build
	./$(BIN_DIR)/$(APP_NAME)

test:
	go test ./... -v

test-short:
	go test ./... -v -short

clean:
	rm -rf $(BIN_DIR)
	rm -f plugins/*/$(shell basename $(PLUGIN_DIR)/*)

plugins:
	@for dir in $(PLUGIN_DIR)/*/; do \
		name=$$(basename $$dir); \
		echo "Building plugin: $$name"; \
		cd $$dir && go build -o $$name . && cd ../..; \
	done

install: build plugins
	mkdir -p ~/.$(APP_NAME)/plugins
	cp $(BIN_DIR)/$(APP_NAME) /usr/local/bin/
	cp -r $(PLUGIN_DIR)/* ~/.$(APP_NAME)/plugins/
	cp -r web ~/.$(APP_NAME)/

lint:
	golangci-lint run

fmt:
	go fmt ./...
```

**Step 2: Verify make works**

Run: `make build`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add Makefile
git commit -m "feat: add Makefile for build/test/install"
```

---

## Summary

This plan covers 20 tasks across 11 phases:

1. **Foundation** - Go module, SQLite store, gesture repository
2. **Camera** - GoCV capture, motion detection
3. **Detection** - MediaPipe interface, mock detector
4. **Matching** - Static (Euclidean) and dynamic (DTW) matchers
5. **Plugins** - Manager, executor, IPC protocol
6. **Server** - HTTP server, REST API
7. **Tray** - macOS menu bar app
8. **Pipeline** - Detection loop with idle/active modes
9. **Bundled Plugins** - system-control, keyboard
10. **Web UI** - Dashboard, gesture list, settings
11. **Polish** - Wiring, Makefile

Each task follows TDD with:
- Failing test first
- Minimal implementation
- Verify passing
- Commit

Total estimated commits: ~25
