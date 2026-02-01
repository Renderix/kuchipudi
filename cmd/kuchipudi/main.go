package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ayusman/kuchipudi/internal/app"
	"github.com/ayusman/kuchipudi/internal/server"
	"github.com/ayusman/kuchipudi/internal/store"
)

func main() {
	fmt.Println("Kuchipudi - Hand Gesture Recognition")

	// Initialize the store
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	dbDir := filepath.Join(homeDir, ".kuchipudi")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dbDir, "kuchipudi.db")
	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer st.Close()

	// Find web directory
	webDir := findWebDir()
	if webDir != "" {
		fmt.Printf("Serving static files from: %s\n", webDir)
	}

	// Create app with camera and detector
	pluginDir := filepath.Join(dbDir, "plugins")
	appCfg := app.Config{
		Store:        st,
		PluginDir:    pluginDir,
		CameraID:     0, // Default camera
		MotionThresh: 0.05,
	}
	application := app.New(appCfg)

	// Load gestures from database
	if err := application.LoadGestures(); err != nil {
		log.Printf("Warning: Failed to load gestures: %v", err)
	}

	// Discover plugins
	if err := application.DiscoverPlugins(); err != nil {
		log.Printf("Warning: Failed to discover plugins: %v", err)
	}

	// NOTE: Don't start app pipeline - let WebSocket handler read camera directly
	// This avoids camera access conflicts between pipeline and recording UI
	// application.SetEnabled(true)
	// if err := application.Start(); err != nil {
	// 	log.Fatalf("Failed to start detection pipeline: %v", err)
	// }
	// defer application.Stop()

	// Configure and start server with app's camera and detector
	cfg := server.Config{
		StaticDir: webDir,
		Store:     st,
		Camera:    application.Camera(),
		Detector:  application.Detector(),
	}

	srv := server.New(cfg)

	addr := ":8080"
	fmt.Printf("Starting server on %s\n", addr)
	fmt.Println("Open http://localhost:8080 in your browser")
	fmt.Println("Press Ctrl+C to stop")

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(addr); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
}

// findWebDir searches for the web directory in common locations.
// It checks: "web", "../web", "../../web", and ~/.kuchipudi/web.
// Returns the first existing directory or empty string if none found.
func findWebDir() string {
	// Check relative paths from current working directory
	relativePaths := []string{"web", "../web", "../../web"}
	for _, p := range relativePaths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			absPath, err := filepath.Abs(p)
			if err == nil {
				return absPath
			}
			return p
		}
	}

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	homeWebDir := filepath.Join(homeDir, ".kuchipudi", "web")
	if info, err := os.Stat(homeWebDir); err == nil && info.IsDir() {
		return homeWebDir
	}

	return ""
}
