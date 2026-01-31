# Kuchipudi - Hand Gesture Recognition System

A lightweight macOS daemon that recognizes hand gestures from webcam input and triggers configurable actions.

## Overview

**Use Cases:**
- System control (volume, brightness, media playback)
- Application shortcuts (launch apps, keyboard shortcuts, scripts)
- Smart home control (via plugin interface, integrations added later)

**Key Requirements:**
- Lowest possible memory footprint and CPU usage for background operation
- WebView configuration UI for gesture management
- User-trainable gestures via video recording
- Hand detection and path tracing for action recognition

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Kuchipudi                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   Capture    │───▶│   Detector   │───▶│   Matcher    │       │
│  │   (Camera)   │    │  (Hybrid ML) │    │  (Gestures)  │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│         │                                        │               │
│         │                                        ▼               │
│         │                               ┌──────────────┐        │
│         │                               │   Executor   │        │
│         │                               │  (Plugins)   │        │
│         │                               └──────────────┘        │
│         │                                        │               │
│         ▼                                        ▼               │
│  ┌──────────────┐                       ┌──────────────┐        │
│  │  Web Server  │◀─────────────────────▶│    Store     │        │
│  │  (Config UI) │                       │  (SQLite)    │        │
│  └──────────────┘                       └──────────────┘        │
│         │                                                        │
├─────────┼────────────────────────────────────────────────────────┤
│         ▼                                                        │
│  ┌──────────────┐                                               │
│  │  Tray Icon   │  ← Enable/Disable, Quick Status               │
│  └──────────────┘                                               │
└─────────────────────────────────────────────────────────────────┘
```

**Core Components:**
- **Capture** - Grabs frames from webcam at low FPS (5-10) when idle
- **Detector** - Motion detection (always on, cheap) triggers ML hand detection (expensive, on-demand)
- **Matcher** - Compares detected hand poses/paths against trained gestures
- **Executor** - Routes matched gestures to appropriate plugin
- **Store** - SQLite for gesture definitions, action mappings, settings (persisted to `~/.kuchipudi/data.db`)
- **Web Server** - Local HTTP server serving the config UI
- **Tray Icon** - macOS menu bar for quick toggle and status

## Hybrid Detection System

The key to low resource usage is only running ML when needed.

### Idle State (~0.5% CPU)
```
Camera (5 FPS) → Motion Detector → No motion? → Sleep 200ms → Loop
```

### Active State (~10-15% CPU, only when gesture detected)
```
Motion detected → Increase to 15 FPS → ML Hand Detection →
Hand found? → Track landmarks → Match gesture → Execute action →
No motion for 2s? → Return to idle
```

### Motion Detection (Pure Go, cheap)
- Compare consecutive frames
- Calculate pixel difference in a region of interest
- Threshold triggers "wake up"
- No external dependencies

### Hand Detection (MediaPipe via CGO)
- Google's MediaPipe Hands model - lightweight, accurate
- Returns 21 hand landmarks (fingertips, knuckles, wrist)
- Called via CGO wrapper to MediaPipe C++ library
- Only loaded into memory when motion detected
- Unloaded after 30s of inactivity to free RAM

### Path Tracking
- Buffer last 60 frames of hand positions (~4 seconds at 15 FPS)
- For static poses: compare landmark positions against templates
- For dynamic gestures: analyze movement trajectory (direction, speed, shape)

## Gesture Learning & Matching

### Recording a New Gesture (via Web UI)

1. User clicks "New Gesture" → enters name (e.g., "swipe-left")
2. Live camera preview shows hand landmarks overlaid
3. User performs gesture 3-5 times, clicking "Capture" each time
4. System extracts features from each recording:
   - **Static:** Landmark positions normalized to hand size
   - **Dynamic:** Path vectors, velocity, direction changes
5. System averages samples to create gesture template
6. User maps gesture to an action (select plugin + configure)

### Gesture Template Storage

```go
type GestureTemplate struct {
    ID          string
    Name        string
    Type        string  // "static" or "dynamic"
    Landmarks   [][]float64  // normalized positions for static
    Path        []PathPoint  // trajectory for dynamic
    Tolerance   float64      // matching sensitivity
    Samples     int          // how many recordings trained it
}
```

### Matching Algorithm
- **Static gestures:** Euclidean distance between current landmarks and template. Below threshold = match.
- **Dynamic gestures:** Dynamic Time Warping (DTW) to compare paths. Handles speed variations (fast swipe vs slow swipe = same gesture).

### Conflict Detection
- When saving a new gesture, system checks similarity against existing gestures
- Warns user if two gestures are too similar to reliably distinguish

## Plugin System

Plugins are standalone executables that communicate via stdin/stdout JSON. This keeps them isolated and language-agnostic.

### Plugin Protocol

```go
// Core sends to plugin
type PluginRequest struct {
    Action  string          `json:"action"`  // "execute", "configure", "validate"
    Gesture string          `json:"gesture"` // which gesture triggered this
    Config  json.RawMessage `json:"config"`  // plugin-specific settings
}

// Plugin responds
type PluginResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
    Data    any    `json:"data,omitempty"`
}
```

### Plugin Discovery
- Plugins live in `~/.kuchipudi/plugins/`
- Each plugin has a manifest file (`plugin.json`):

```json
{
    "name": "system-control",
    "version": "1.0.0",
    "executable": "system-control",
    "actions": ["volume-up", "volume-down", "brightness", "media-play-pause"],
    "configSchema": { ... }
}
```

### Bundled Plugins (V1)

| Plugin | Actions |
|--------|---------|
| `system-control` | Volume, brightness, media keys |
| `app-launcher` | Open apps, run shell commands |
| `keyboard` | Send keystrokes, shortcuts |

### Future Plugins (user or community)
- `homeassistant` - Smart home control
- `spotify` - Specific Spotify controls
- `obs` - Streaming scene switching

Plugins run as child processes, spawned on-demand, killed after 30s idle to save resources.

## Web UI & Tray App

### Tray App (Menu Bar)

```
┌─────────────────────────┐
│ 👋 Kuchipudi            │
├─────────────────────────┤
│ ● Enabled               │  ← Toggle on/off
│ ─────────────────────── │
│ Last: swipe-left (2s)   │  ← Recent gesture
│ ─────────────────────── │
│ Open Settings...        │  ← Opens browser to localhost
│ ─────────────────────── │
│ Quit                    │
└─────────────────────────┘
```

### Web UI Pages

| Page | Purpose |
|------|---------|
| Dashboard | Status, recent gestures, quick enable/disable |
| Gestures | List all gestures, add/edit/delete, test matching |
| Record | Live camera feed, record new gesture samples |
| Actions | Map gestures to plugins, configure plugin settings |
| Plugins | View installed plugins, enable/disable |
| Settings | Camera selection, sensitivity, startup behavior |

### Tech Stack
- Backend: Go (net/http, serves static files + JSON API)
- Frontend: Vanilla HTML/CSS/JS (no framework, keeps it light)
- Camera preview: WebRTC or MJPEG stream from Go backend
- Storage API: REST endpoints for CRUD on gestures/actions

### Local Only
- Binds to `127.0.0.1:9847` (not exposed to network)
- No authentication needed (local access only)

## Project Structure

```
kuchipudi/
├── cmd/
│   └── kuchipudi/
│       └── main.go              # Entry point
├── internal/
│   ├── capture/
│   │   ├── camera.go            # AVFoundation webcam capture
│   │   └── motion.go            # Motion detection
│   ├── detector/
│   │   ├── mediapipe.go         # CGO wrapper for MediaPipe
│   │   └── landmarks.go         # Hand landmark types
│   ├── gesture/
│   │   ├── template.go          # Gesture template storage
│   │   ├── matcher.go           # Static + DTW matching
│   │   └── recorder.go          # Training sample collection
│   ├── plugin/
│   │   ├── manager.go           # Plugin discovery, lifecycle
│   │   ├── executor.go          # IPC with plugin processes
│   │   └── schema.go            # Plugin manifest parsing
│   ├── store/
│   │   ├── sqlite.go            # Database operations
│   │   └── migrations/          # Schema migrations
│   ├── server/
│   │   ├── server.go            # HTTP server
│   │   ├── api/                  # REST handlers
│   │   └── stream.go            # Camera feed streaming
│   └── tray/
│       └── tray_darwin.go       # macOS menu bar
├── plugins/
│   ├── system-control/          # Bundled plugin
│   ├── app-launcher/            # Bundled plugin
│   └── keyboard/                # Bundled plugin
├── web/
│   ├── index.html
│   ├── css/
│   └── js/
├── go.mod
├── go.sum
└── Makefile
```

### Key Dependencies
- `gocv.io/x/gocv` - OpenCV bindings for camera + motion detection
- MediaPipe Go wrapper or CGO bindings - Hand detection
- `github.com/getlantern/systray` - macOS tray icon
- `modernc.org/sqlite` - Pure Go SQLite (no CGO for DB)

## Resource Targets

| State | CPU | RAM | Notes |
|-------|-----|-----|-------|
| Idle (no motion) | <1% | ~50MB | Motion detection only |
| Active (gesture detected) | 10-15% | ~150MB | MediaPipe loaded |
| Web UI open | +2% | +20MB | Streaming camera feed |

## Startup Behavior
- Launch at login (optional, configurable)
- Start in idle state immediately
- Load plugins on-demand, not at startup
- MediaPipe model loaded on first motion detection

## Build & Distribution

```makefile
build:
    go build -ldflags="-s -w" -o bin/kuchipudi ./cmd/kuchipudi

install:
    cp bin/kuchipudi /usr/local/bin/
    cp -r plugins/ ~/.kuchipudi/plugins/

bundle:  # Create .app bundle for macOS
    # Package as Kuchipudi.app with proper Info.plist
    # Sign for Gatekeeper (optional, for distribution)
```

## Permissions Required (macOS)
- Camera access (prompted on first run)
- Accessibility access (for keyboard/system control plugins)

## Platform
- macOS only for V1
- Clean interfaces designed for future portability to Linux/Windows
