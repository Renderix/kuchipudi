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
	"time"

	"gocv.io/x/gocv"
)

// MediaPipeDetector implements Detector using a Python MediaPipe subprocess.
type MediaPipeDetector struct {
	config    Config
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	mu        sync.Mutex
	started   bool
	lastUsed  time.Time
	idleTimer *time.Timer
}

// NewMediaPipeDetector creates a new MediaPipe detector.
// The Python process is started lazily on first detection.
func NewMediaPipeDetector(config Config) (*MediaPipeDetector, error) {
	scriptPath := findMediaPipeScript()
	if scriptPath == "" {
		return nil, fmt.Errorf("mediapipe_service.py not found")
	}

	return &MediaPipeDetector{
		config: config,
	}, nil
}

// Detect analyzes a frame and returns detected hand landmarks.
func (d *MediaPipeDetector) Detect(frame *gocv.Mat) ([]HandLandmarks, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.ensureStarted(); err != nil {
		return nil, err
	}

	// Encode frame as JPEG
	buf, err := gocv.IMEncode(".jpg", *frame)
	if err != nil {
		return nil, fmt.Errorf("encode frame: %w", err)
	}
	defer buf.Close()

	data := buf.GetBytes()

	// Write length (4 bytes big-endian) + data
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data)))

	if _, err := d.stdin.Write(length); err != nil {
		return nil, fmt.Errorf("write length: %w", err)
	}
	if _, err := d.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write data: %w", err)
	}

	// Read JSON response
	line, err := d.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var response struct {
		Hands []jsonHand `json:"hands"`
	}
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Convert to HandLandmarks
	result := make([]HandLandmarks, len(response.Hands))
	for i, h := range response.Hands {
		result[i] = h.toHandLandmarks()
	}

	d.lastUsed = time.Now()
	d.resetIdleTimer()

	return result, nil
}

// Close shuts down the Python process.
func (d *MediaPipeDetector) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.shutdown()
}

func (d *MediaPipeDetector) ensureStarted() error {
	if d.started {
		return nil
	}

	scriptPath := findMediaPipeScript()
	if scriptPath == "" {
		return fmt.Errorf("mediapipe_service.py not found")
	}

	// Use virtual environment Python if available
	pythonPath := findVenvPython()
	if pythonPath == "" {
		pythonPath = "python3"
	}

	d.cmd = exec.Command(pythonPath, scriptPath)

	stdin, err := d.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	// Capture stderr for debugging
	d.cmd.Stderr = os.Stderr

	if err := d.cmd.Start(); err != nil {
		return fmt.Errorf("start mediapipe service: %w", err)
	}

	d.stdin = stdin
	d.stdout = bufio.NewReader(stdout)
	d.started = true
	d.lastUsed = time.Now()

	return nil
}

func (d *MediaPipeDetector) shutdown() error {
	if !d.started {
		return nil
	}

	if d.idleTimer != nil {
		d.idleTimer.Stop()
		d.idleTimer = nil
	}

	if d.stdin != nil {
		d.stdin.Close()
	}

	err := d.cmd.Wait()
	d.started = false
	d.cmd = nil
	d.stdin = nil
	d.stdout = nil

	return err
}

func (d *MediaPipeDetector) resetIdleTimer() {
	if d.idleTimer != nil {
		d.idleTimer.Stop()
	}
	d.idleTimer = time.AfterFunc(30*time.Second, func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.shutdown()
	})
}

func findMediaPipeScript() string {
	// Get executable directory
	execPath, err := os.Executable()
	var execDir string
	if err == nil {
		execDir = filepath.Dir(execPath)
	}

	candidates := []string{
		"scripts/mediapipe_service.py",
		"../scripts/mediapipe_service.py",
		filepath.Join(execDir, "scripts/mediapipe_service.py"),
		filepath.Join(os.Getenv("HOME"), ".kuchipudi/scripts/mediapipe_service.py"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				return absPath
			}
			return path
		}
	}
	return ""
}

// findVenvPython looks for a Python interpreter in a virtual environment.
// It checks for venv/bin/python relative to the project directory.
func findVenvPython() string {
	// Get executable directory to find project root
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	execDir := filepath.Dir(execPath)

	candidates := []string{
		"venv/bin/python",
		"../venv/bin/python",
		"../../venv/bin/python",
		filepath.Join(execDir, "venv/bin/python"),
		filepath.Join(os.Getenv("HOME"), ".kuchipudi/venv/bin/python"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				return absPath
			}
			return path
		}
	}
	return ""
}

// jsonHand represents the JSON structure from the Python service.
type jsonHand struct {
	Points     []jsonPoint `json:"points"`
	Handedness string      `json:"handedness"`
	Score      float64     `json:"score"`
}

type jsonPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (h jsonHand) toHandLandmarks() HandLandmarks {
	lm := HandLandmarks{
		Handedness: h.Handedness,
		Score:      h.Score,
	}

	for i := 0; i < NumLandmarks && i < len(h.Points); i++ {
		lm.Points[i] = Point3D{
			X: h.Points[i].X,
			Y: h.Points[i].Y,
			Z: h.Points[i].Z,
		}
	}

	return lm
}
