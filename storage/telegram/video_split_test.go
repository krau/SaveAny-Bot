package telegram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitialSegmentDuration(t *testing.T) {
	tests := []struct {
		name       string
		duration   float64
		fileSize   int64
		targetSize int64
		want       float64
	}{
		{name: "two balanced parts", duration: 120, fileSize: 2200, targetSize: 1900, want: 60},
		{name: "three balanced parts", duration: 120, fileSize: 3900, targetSize: 1900, want: 40},
		{name: "minimum one second", duration: 1.5, fileSize: 3900, targetSize: 1900, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := initialSegmentDuration(tt.duration, tt.fileSize, tt.targetSize)
			if got != tt.want {
				t.Fatalf("initialSegmentDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVideoPartExtension(t *testing.T) {
	tests := map[string]string{
		"clip.MOV": ".mov",
		"clip.mkv": ".mkv",
		"clip.exe": ".mp4",
		"clip":     ".mp4",
	}
	for input, want := range tests {
		if got := videoPartExtension(input); got != want {
			t.Fatalf("videoPartExtension(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPartStoragePath(t *testing.T) {
	tests := map[string]string{
		"clip.mov":            "clip.part001.mov",
		"123456/clip.mov":     "123456/clip.part001.mov",
		"/123456/clip.mov":    "/123456/clip.part001.mov",
		"folder/sub/clip.mov": "folder/sub/clip.part001.mov",
	}
	for input, want := range tests {
		if got := partStoragePath(input, "clip.part001.mov"); got != want {
			t.Fatalf("partStoragePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestVideoPartCaption(t *testing.T) {
	if got := videoPartCaption(nil, 0); got != nil {
		t.Fatalf("caption without source = %q, want nil", *got)
	}

	source := "original caption"
	first := videoPartCaption(&source, 0)
	if first == nil || *first != source {
		t.Fatalf("first part caption = %v, want %q", first, source)
	}
	second := videoPartCaption(&source, 1)
	if second == nil || *second != "" {
		t.Fatalf("second part caption = %v, want an explicit empty caption", second)
	}
}

func TestSplitLosslessVideoRetriesOversizedPart(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "source.mov")
	if err := os.WriteFile(inputPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(tempDir, "parts")
	if err := os.Mkdir(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalRunner := runMediaTool
	t.Cleanup(func() { runMediaTool = originalRunner })
	ffmpegCalls := 0
	runMediaTool = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "ffprobe":
			return []byte("120.0\n"), nil
		case "ffmpeg":
			ffmpegCalls++
			pattern := args[len(args)-1]
			sizes := []int{2001, 199}
			if ffmpegCalls > 1 {
				sizes = []int{1100, 1100}
			}
			for index, size := range sizes {
				partPath := strings.Replace(pattern, "%03d", fmt.Sprintf("%03d", index+1), 1)
				if err := os.WriteFile(partPath, make([]byte, size), 0o600); err != nil {
					return nil, err
				}
			}
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected media tool %q", name)
		}
	}

	parts, err := splitLosslessVideo(t.Context(), inputPath, outputDir, "example.mov", 2200, 2000)
	if err != nil {
		t.Fatalf("splitLosslessVideo() failed: %v", err)
	}
	if ffmpegCalls != 2 {
		t.Fatalf("ffmpeg calls = %d, want 2", ffmpegCalls)
	}
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want 2", len(parts))
	}
	if parts[0].Name != "example.part001.mov" || parts[1].Name != "example.part002.mov" {
		t.Fatalf("unexpected part names: %#v", parts)
	}
	for _, part := range parts {
		if part.Size > 2000 {
			t.Fatalf("part %s exceeds limit: %d", part.Name, part.Size)
		}
	}
}

func TestSplitLosslessVideoRejectsMultipleAlbums(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "source.mov")
	if err := os.WriteFile(inputPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(tempDir, "parts")
	if err := os.Mkdir(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	originalRunner := runMediaTool
	t.Cleanup(func() { runMediaTool = originalRunner })
	runMediaTool = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "ffprobe":
			return []byte("120.0\n"), nil
		case "ffmpeg":
			pattern := args[len(args)-1]
			for index := 0; index < maxLosslessVideoParts+1; index++ {
				partPath := strings.Replace(pattern, "%03d", fmt.Sprintf("%03d", index+1), 1)
				if err := os.WriteFile(partPath, make([]byte, 100), 0o600); err != nil {
					return nil, err
				}
			}
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected media tool %q", name)
		}
	}

	_, err := splitLosslessVideo(t.Context(), inputPath, outputDir, "example.mov", 2200, 2000)
	if err == nil {
		t.Fatal("splitLosslessVideo() succeeded with more than one album of parts")
	}
	if !strings.Contains(err.Error(), "single-album limit") {
		t.Fatalf("splitLosslessVideo() error = %q, want single-album limit", err)
	}
}
