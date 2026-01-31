# Kuchipudi 👋

A lightweight macOS hand gesture recognition system that detects hand gestures from your webcam and triggers configurable actions.

Named after [Kuchipudi](https://en.wikipedia.org/wiki/Kuchipudi), a classical Indian dance form known for its expressive hand gestures.

## Features

- **Low Resource Usage**: Hybrid detection system keeps CPU <1% when idle
- **Trainable Gestures**: Record custom gestures with 3-5 samples
- **Plugin System**: Extensible action system with bundled plugins
- **Web Configuration**: Browser-based UI for gesture management
- **Menu Bar App**: Quick toggle and status in macOS menu bar

## Quick Start

### Prerequisites

- macOS 12.0 or later
- Go 1.21+
- OpenCV 4.x (`brew install opencv`)
- Python 3.8+ with MediaPipe (optional, for hand detection)

### Installation

```bash
# Clone the repository
git clone https://github.com/ayusman/kuchipudi.git
cd kuchipudi

# Build everything
make build
make plugins

# Install (optional)
make install
```

### Running

```bash
# Run directly
./bin/kuchipudi

# Or if installed
kuchipudi
```

The app will:
1. Start in the menu bar with a 👋 icon
2. Open a local web server at http://127.0.0.1:9847
3. Begin monitoring for hand gestures

## Usage

### Menu Bar

Click the 👋 icon in your menu bar to:
- **Enable/Disable**: Toggle gesture recognition on/off
- **Open Settings**: Launch the web configuration UI
- **Quit**: Exit the application

### Web Interface

Open http://127.0.0.1:9847 in your browser to:

- **Dashboard**: View status and recent gesture activity
- **Gestures**: Manage your trained gestures
- **Actions**: Map gestures to plugin actions
- **Settings**: Configure camera and sensitivity

### Recording a Gesture

1. Click "Add Gesture" in the Gestures page
2. Enter a name (e.g., "thumbs-up")
3. Select type:
   - **Static**: A held pose (like thumbs up)
   - **Dynamic**: A movement (like swipe left)
4. Record 3-5 samples by performing the gesture
5. Click Save

### Mapping Actions

1. Go to the Actions page
2. Select a gesture from the dropdown
3. Choose a plugin and action
4. Configure any action-specific settings
5. Click Save

## Bundled Plugins

### system-control

Control macOS system settings:

| Action | Description |
|--------|-------------|
| `volume-up` | Increase volume by 10% |
| `volume-down` | Decrease volume by 10% |
| `volume-mute` | Toggle mute |
| `brightness-up` | Increase brightness |
| `brightness-down` | Decrease brightness |
| `media-play-pause` | Play/pause media |
| `media-next` | Next track |
| `media-prev` | Previous track |

### keyboard

Send keyboard shortcuts:

| Action | Parameters | Example |
|--------|------------|---------|
| `keystroke` | `key`, `modifiers` | `{"key": "c", "modifiers": ["command"]}` |
| `shortcut` | Same as keystroke | Copy shortcut |

Supported modifiers: `command`, `option`, `control`, `shift`

## Configuration

### Data Directory

All data is stored in `~/.kuchipudi/`:

```
~/.kuchipudi/
├── data.db          # SQLite database (gestures, actions, settings)
├── plugins/         # Installed plugins
│   ├── system-control/
│   └── keyboard/
└── web/             # Web UI files (if installed)
```

### Settings

| Setting | Default | Description |
|---------|---------|-------------|
| Camera | 0 | Camera device ID |
| Motion Threshold | 5% | Pixel change % to trigger detection |
| Idle FPS | 5 | Frame rate when no motion |
| Active FPS | 15 | Frame rate during gesture detection |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Kuchipudi                                │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   Camera     │───▶│   Motion     │───▶│    Hand      │       │
│  │   Capture    │    │   Detector   │    │   Detector   │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│         │                                        │               │
│         │                                        ▼               │
│         │                               ┌──────────────┐        │
│         │                               │   Gesture    │        │
│         │                               │   Matcher    │        │
│         │                               └──────────────┘        │
│         │                                        │               │
│         ▼                                        ▼               │
│  ┌──────────────┐                       ┌──────────────┐        │
│  │  Web Server  │                       │   Plugin     │        │
│  │  (Config UI) │                       │   Executor   │        │
│  └──────────────┘                       └──────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

**Detection Flow:**
1. Camera captures frames at low FPS (idle mode)
2. Motion detector looks for pixel changes
3. On motion, switches to high FPS and runs hand detection
4. Hand landmarks are matched against trained gestures
5. Matched gestures trigger configured plugin actions
6. After 2s of no motion, returns to idle mode

## Development

### Building

```bash
# Build main app
make build

# Build plugins
make plugins

# Build everything
make build && make plugins
```

### Testing

```bash
# Run all tests
make test

# Run tests without CGO (faster)
make test-short

# Run specific package tests
go test ./internal/gesture/... -v
```

### Project Structure

```
kuchipudi/
├── cmd/kuchipudi/       # Main application entry point
├── internal/
│   ├── app/             # Application orchestrator
│   ├── capture/         # Camera and motion detection
│   ├── detector/        # Hand detection interface
│   ├── gesture/         # Gesture matching (static + DTW)
│   ├── plugin/          # Plugin manager and executor
│   ├── server/          # HTTP server and API
│   ├── store/           # SQLite database
│   └── tray/            # macOS menu bar
├── plugins/
│   ├── system-control/  # System control plugin
│   └── keyboard/        # Keyboard shortcut plugin
├── web/                 # Web UI (HTML/CSS/JS)
└── docs/plans/          # Implementation plans
```

### Creating a Plugin

1. Create a directory in `plugins/`:
   ```
   plugins/my-plugin/
   ├── plugin.json
   └── main.go
   ```

2. Write `plugin.json`:
   ```json
   {
       "name": "my-plugin",
       "version": "1.0.0",
       "description": "My custom plugin",
       "executable": "my-plugin",
       "actions": ["action1", "action2"]
   }
   ```

3. Write `main.go`:
   ```go
   package main

   import (
       "encoding/json"
       "os"
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
       json.NewDecoder(os.Stdin).Decode(&req)

       // Handle action
       resp := Response{Success: true}
       json.NewEncoder(os.Stdout).Encode(resp)
   }
   ```

4. Build:
   ```bash
   cd plugins/my-plugin && go build -o my-plugin .
   ```

## Troubleshooting

### Camera not detected

- Grant camera permission in System Preferences > Privacy & Security > Camera
- Try a different camera ID in settings

### Hand detection not working

- Ensure good lighting
- Position your hand clearly in the frame
- Check that MediaPipe is installed: `pip install mediapipe`

### Plugin not executing

- Check plugin is built: `ls ~/.kuchipudi/plugins/*/`
- Grant Accessibility permission for keyboard/system control plugins

### High CPU usage

- Reduce Active FPS in settings
- Increase Motion Threshold to reduce false activations

## Permissions Required

- **Camera**: For capturing video frames
- **Accessibility**: For keyboard and system control plugins (grant in System Preferences > Privacy & Security > Accessibility)

## License

MIT

## Acknowledgments

- [MediaPipe](https://mediapipe.dev/) - Hand detection model
- [GoCV](https://gocv.io/) - OpenCV bindings for Go
- [systray](https://github.com/getlantern/systray) - Cross-platform system tray
