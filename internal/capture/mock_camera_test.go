package capture

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestMockCamera_Playback(t *testing.T) {
	// Create test frames
	frame1 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame1.Close()
	frame2 := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame2.Close()

	cam := NewMockCamera([]*gocv.Mat{&frame1, &frame2}, false)

	if err := cam.Open(); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer cam.Close()

	// Read both frames
	f1, err := cam.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	f1.Close()

	f2, err := cam.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	f2.Close()

	// Third read should fail (no loop)
	_, err = cam.ReadFrame()
	if err == nil {
		t.Error("expected error after all frames consumed")
	}
}

func TestMockCamera_Loop(t *testing.T) {
	frame := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8UC3)
	defer frame.Close()

	cam := NewMockCamera([]*gocv.Mat{&frame}, true)
	cam.Open()
	defer cam.Close()

	// Should loop indefinitely
	for i := 0; i < 5; i++ {
		f, err := cam.ReadFrame()
		if err != nil {
			t.Fatalf("ReadFrame() iteration %d error = %v", i, err)
		}
		f.Close()
	}
}
