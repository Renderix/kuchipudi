package app

import (
	"log"
	"time"

	"github.com/ayusman/kuchipudi/internal/gesture"
	"github.com/ayusman/kuchipudi/internal/plugin"
)

// runPipeline is the main detection loop that processes frames from the camera.
// It manages the state transitions between idle and active modes based on motion detection.
//
// Pipeline logic:
// 1. Start in idle mode (idleFPS=5)
// 2. On motion detected, switch to active mode (activeFPS=15)
// 3. Run hand detection
// 4. Match against static/dynamic gestures
// 5. Buffer path for dynamic gestures (last 60 frames)
// 6. After 2s no motion, switch back to idle mode
// 7. Clear path buffer on dynamic match to prevent repeated triggers
func (a *App) runPipeline() {
	// Path buffer for dynamic gesture detection
	pathBuffer := make([]gesture.PathPoint, 0, PathBufferSize)

	// Track whether we're in active mode
	activeMode := false

	// Track the last motion detection time
	lastMotionTime := time.Now()

	// Frame interval based on current FPS
	frameInterval := time.Second / time.Duration(IdleFPS)

	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			// Skip processing if detection is disabled
			if !a.IsEnabled() {
				continue
			}

			// Read a frame from the camera
			frame, err := a.camera.ReadFrame()
			if err != nil {
				log.Printf("Error reading frame: %v", err)
				continue
			}

			// Step 1: Motion detection
			motionDetected, _ := a.motion.Detect(frame)

			if motionDetected {
				lastMotionTime = time.Now()

				// Switch to active mode if not already
				if !activeMode {
					activeMode = true
					a.camera.SetFPS(ActiveFPS)
					frameInterval = time.Second / time.Duration(ActiveFPS)
					ticker.Reset(frameInterval)
					log.Println("Switched to active mode")
				}
			} else if activeMode {
				// Check if we should switch back to idle mode
				if time.Since(lastMotionTime) > time.Duration(IdleTimeoutMs)*time.Millisecond {
					activeMode = false
					a.camera.SetFPS(IdleFPS)
					frameInterval = time.Second / time.Duration(IdleFPS)
					ticker.Reset(frameInterval)
					pathBuffer = pathBuffer[:0] // Clear path buffer
					log.Println("Switched to idle mode")
				}
			}

			// Skip further processing if not in active mode or no detector
			if !activeMode || a.detector == nil {
				frame.Close()
				continue
			}

			// Step 2: Hand detection
			hands, err := a.detector.Detect(frame)
			frame.Close() // Done with the frame

			if err != nil {
				log.Printf("Error detecting hands: %v", err)
				continue
			}

			if len(hands) == 0 {
				continue
			}

			// Process each detected hand
			for i := range hands {
				hand := &hands[i]

				// Step 3: Static gesture matching
				staticMatches := a.staticMatcher.Match(hand)
				if len(staticMatches) > 0 {
					best := staticMatches[0]
					log.Printf("Static gesture matched: %s (score: %.3f)", best.Template.Name, best.Score)
					a.executeAction(best.Template.ID, best.Template.Name)
				}

				// Step 4: Buffer path for dynamic gesture detection
				// Use the index finger tip position for tracking
				indexTip := hand.Points[8] // IndexTip = 8
				pathPoint := gesture.PathPoint{
					X:         indexTip.X,
					Y:         indexTip.Y,
					Timestamp: time.Now().UnixMilli(),
				}

				// Add to path buffer
				if len(pathBuffer) >= PathBufferSize {
					// Shift buffer left by 1, removing oldest point
					copy(pathBuffer, pathBuffer[1:])
					pathBuffer = pathBuffer[:PathBufferSize-1]
				}
				pathBuffer = append(pathBuffer, pathPoint)

				// Step 5: Dynamic gesture matching (need at least some points)
				if len(pathBuffer) >= 10 {
					dynamicMatches := a.dynamicMatcher.Match(pathBuffer)
					if len(dynamicMatches) > 0 {
						best := dynamicMatches[0]
						log.Printf("Dynamic gesture matched: %s (score: %.3f)", best.Template.Name, best.Score)
						a.executeAction(best.Template.ID, best.Template.Name)

						// Clear path buffer to prevent repeated triggers
						pathBuffer = pathBuffer[:0]
					}
				}
			}
		}
	}
}

// executeAction executes the action associated with a recognized gesture.
// It looks up the action binding in the database and executes the corresponding plugin.
func (a *App) executeAction(gestureID, gestureName string) {
	// Skip if no store configured
	if a.config.Store == nil {
		return
	}

	// Look up action binding
	action, err := a.config.Store.Actions().GetByGestureID(gestureID)
	if err != nil {
		log.Printf("Error looking up action: %v", err)
		return
	}
	if action == nil || !action.Enabled {
		return // No action bound or disabled - silent skip
	}

	// Get plugin
	plug, err := a.pluginMgr.Get(action.PluginName)
	if err != nil {
		log.Printf("Plugin not found: %s", action.PluginName)
		return
	}

	// Build request
	req := &plugin.Request{
		Action:  action.ActionName,
		Gesture: gestureName,
		Config:  action.Config,
	}

	// Execute async to not block pipeline
	go func() {
		resp, err := a.pluginExec.Execute(plug, req)
		if err != nil {
			log.Printf("Plugin execution failed: %v", err)
			return
		}
		if !resp.Success {
			log.Printf("Plugin returned error: %s", resp.Error)
		}
	}()
}
