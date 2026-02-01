// Package app provides the main application logic for the Kuchipudi gesture recognition system.
package app

import (
	"log"
	"sync"
	"time"

	"github.com/ayusman/kuchipudi/internal/capture"
	"github.com/ayusman/kuchipudi/internal/detector"
	"github.com/ayusman/kuchipudi/internal/gesture"
	"github.com/ayusman/kuchipudi/internal/plugin"
	"github.com/ayusman/kuchipudi/internal/store"
)

// Pipeline timing constants.
const (
	// IdleFPS is the frame rate when no motion is detected.
	IdleFPS = 5
	// ActiveFPS is the frame rate during active detection.
	ActiveFPS = 15
	// IdleTimeoutMs is the time in milliseconds to wait before switching back to idle mode.
	IdleTimeoutMs = 2000
	// PathBufferSize is the maximum number of frames to buffer for dynamic gesture detection.
	PathBufferSize = 60
)

// Config holds configuration options for the application.
type Config struct {
	Store        *store.Store
	PluginDir    string
	CameraID     int
	MotionThresh float64
}

// App is the main application that orchestrates gesture detection and action execution.
type App struct {
	config         Config
	camera         capture.Camera
	motion         *capture.MotionDetector
	detector       detector.Detector
	staticMatcher  *gesture.StaticMatcher
	dynamicMatcher *gesture.DynamicMatcher
	pluginMgr      *plugin.Manager
	pluginExec     *plugin.Executor
	enabled        bool
	mu             sync.RWMutex
	stopCh         chan struct{}
	lastMotionTime time.Time
}

// New creates a new App instance with the given configuration.
func New(config Config) *App {
	motionThreshold := config.MotionThresh
	if motionThreshold <= 0 {
		motionThreshold = 1.0 // Default threshold: 1% pixel change
	}

	a := &App{
		config:         config,
		camera:         capture.NewCamera(config.CameraID),
		motion:         capture.NewMotionDetector(motionThreshold),
		staticMatcher:  gesture.NewStaticMatcher(),
		dynamicMatcher: gesture.NewDynamicMatcher(),
		pluginMgr:      plugin.NewManager(config.PluginDir),
		pluginExec:     plugin.NewExecutor(5000), // 5 second timeout for plugin execution
		enabled:        false,
		stopCh:         nil,
		lastMotionTime: time.Now(),
	}

	// Try MediaPipe first, fall back to mock detector
	if mp, err := detector.NewMediaPipeDetector(detector.DefaultConfig()); err == nil {
		a.detector = mp
		log.Println("Using MediaPipe hand detection")
	} else {
		log.Printf("MediaPipe not available (%v), using mock detector", err)
		a.detector = detector.NewMockDetector()
	}

	return a
}

// SetEnabled enables or disables gesture detection.
func (a *App) SetEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = enabled
}

// IsEnabled returns whether gesture detection is currently enabled.
func (a *App) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// SetDetector sets the hand detector implementation to use.
func (a *App) SetDetector(d detector.Detector) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.detector = d
}

// LoadGestures loads gesture templates from the database into the matchers.
func (a *App) LoadGestures() error {
	if a.config.Store == nil {
		return nil
	}

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

		switch g.Type {
		case store.GestureTypeStatic:
			template.Type = gesture.TypeStatic
			landmarks, err := a.config.Store.Gestures().GetLandmarks(g.ID)
			if err != nil {
				log.Printf("Failed to load landmarks for %s: %v", g.Name, err)
			} else if len(landmarks) > 0 {
				template.Landmarks = storeLandmarksToDetector(landmarks)
			}
			a.staticMatcher.AddTemplate(template)

		case store.GestureTypeDynamic:
			template.Type = gesture.TypeDynamic
			path, err := a.config.Store.Gestures().GetPath(g.ID)
			if err != nil {
				log.Printf("Failed to load path for %s: %v", g.Name, err)
			} else if len(path) > 0 {
				template.Path = storePathToGesture(path)
			}
			a.dynamicMatcher.AddTemplate(template)
		}
	}

	log.Printf("Loaded %d gestures from database", len(gestures))
	return nil
}

// storeLandmarksToDetector converts store.Landmark slice to detector.Point3D slice.
func storeLandmarksToDetector(landmarks []store.Landmark) []detector.Point3D {
	points := make([]detector.Point3D, len(landmarks))
	for i, l := range landmarks {
		points[i] = detector.Point3D{X: l.X, Y: l.Y, Z: l.Z}
	}
	return points
}

// storePathToGesture converts store.PathPoint slice to gesture.PathPoint slice.
func storePathToGesture(path []store.PathPoint) []gesture.PathPoint {
	points := make([]gesture.PathPoint, len(path))
	for i, p := range path {
		points[i] = gesture.PathPoint{X: p.X, Y: p.Y, Timestamp: p.TimestampMs}
	}
	return points
}

// DiscoverPlugins scans the plugin directory and loads available plugins.
func (a *App) DiscoverPlugins() error {
	return a.pluginMgr.Discover()
}

// Start begins the detection pipeline.
func (a *App) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Don't start if already running
	if a.stopCh != nil {
		return nil
	}

	// Open the camera
	if err := a.camera.Open(); err != nil {
		return err
	}

	// Set initial FPS to idle mode
	a.camera.SetFPS(IdleFPS)

	// Create stop channel and start the pipeline
	a.stopCh = make(chan struct{})
	go a.runPipeline()

	log.Println("Detection pipeline started")
	return nil
}

// Stop halts the detection pipeline and releases resources.
func (a *App) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Signal the pipeline to stop
	if a.stopCh != nil {
		close(a.stopCh)
		a.stopCh = nil
	}

	// Close the camera
	if err := a.camera.Close(); err != nil {
		log.Printf("Error closing camera: %v", err)
	}

	// Close motion detector
	a.motion.Close()

	// Close the hand detector if set
	if a.detector != nil {
		if err := a.detector.Close(); err != nil {
			log.Printf("Error closing detector: %v", err)
		}
	}

	log.Println("Detection pipeline stopped")
}

// Camera returns the camera instance.
func (a *App) Camera() capture.Camera {
	return a.camera
}

// MotionDetector returns the motion detector instance.
func (a *App) MotionDetector() *capture.MotionDetector {
	return a.motion
}

// StaticMatcher returns the static gesture matcher.
func (a *App) StaticMatcher() *gesture.StaticMatcher {
	return a.staticMatcher
}

// DynamicMatcher returns the dynamic gesture matcher.
func (a *App) DynamicMatcher() *gesture.DynamicMatcher {
	return a.dynamicMatcher
}

// PluginManager returns the plugin manager.
func (a *App) PluginManager() *plugin.Manager {
	return a.pluginMgr
}

// Detector returns the hand detector.
func (a *App) Detector() detector.Detector {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.detector
}
