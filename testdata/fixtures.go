package testdata

import (
	"embed"
	"fmt"

	"gocv.io/x/gocv"
)

//go:embed frames/*
var framesFS embed.FS

// LoadFrame loads a test frame by name
func LoadFrame(name string) (*gocv.Mat, error) {
	data, err := framesFS.ReadFile("frames/" + name)
	if err != nil {
		return nil, fmt.Errorf("load frame %s: %w", name, err)
	}

	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("decode frame %s: %w", name, err)
	}

	return &mat, nil
}

// LoadSequence loads a sequence of frames for dynamic gesture testing
func LoadSequence(dir string) ([]*gocv.Mat, error) {
	entries, err := framesFS.ReadDir("frames/" + dir)
	if err != nil {
		return nil, err
	}

	var frames []*gocv.Mat
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		frame, err := LoadFrame(dir + "/" + entry.Name())
		if err != nil {
			// Clean up already loaded frames
			for _, f := range frames {
				f.Close()
			}
			return nil, err
		}
		frames = append(frames, frame)
	}

	return frames, nil
}
