package telegram

import (
	"os"
	"testing"
)

func TestExtractThumbFrame(t *testing.T) {
	file, err := os.Open("tests/testvideo")
	if err != nil {
		t.Fatalf("failed to open test video: %v", err)
	}
	defer file.Close()
	thumb, err := extractThumbFrame(file)
	if err != nil {
		t.Fatalf("failed to extract thumb frame: %v", err)
	}
	os.WriteFile("tests/testthumb.jpg", thumb, 0644)
}

func TestGetVideoMetadata(t *testing.T) {
	file, err := os.Open("tests/testvideo")
	if err != nil {
		t.Fatalf("failed to open test video: %v", err)
	}
	defer file.Close()
	meta, err := getVideoMetadata(file)
	if err != nil {
		t.Fatalf("failed to get video metadata: %v", err)
	}
	if meta.Duration == 0 || meta.Width == 0 || meta.Height == 0 {
		t.Fatalf("invalid video metadata: %+v", meta)
	}
}
