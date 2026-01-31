package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	// Configure and start server
	cfg := server.Config{
		StaticDir: webDir,
		Store:     st,
	}

	srv := server.New(cfg)

	addr := ":8080"
	fmt.Printf("Starting server on %s\n", addr)
	if err := srv.ListenAndServe(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
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
