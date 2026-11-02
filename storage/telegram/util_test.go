package telegram

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func makeTestVideo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	path := filepath.Join(t.TempDir(), "testvideo.mp4")
	cmd := exec.CommandContext(t.Context(), "ffmpeg",
		"-loglevel", "error",
		"-f", "lavfi",
		"-i", "color=c=blue:s=64x64:d=1",
		"-c:v", "mpeg4",
		"-pix_fmt", "yuv420p",
		"-y", path,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create test video: %v: %s", err, output)
	}
	return path
}

func TestExtractThumbFrame(t *testing.T) {
	file, err := os.Open(makeTestVideo(t))
	if err != nil {
		t.Fatalf("failed to open test video: %v", err)
	}
	defer closeTestFile(t, file)
	thumb, err := extractThumbFrame(file)
	if err != nil {
		t.Fatalf("failed to extract thumb frame: %v", err)
	}
	if len(thumb) == 0 {
		t.Fatal("extracted thumbnail is empty")
	}
}

func TestGetVideoMetadata(t *testing.T) {
	file, err := os.Open(makeTestVideo(t))
	if err != nil {
		t.Fatalf("failed to open test video: %v", err)
	}
	defer closeTestFile(t, file)
	meta, err := getVideoMetadata(file)
	if err != nil {
		t.Fatalf("failed to get video metadata: %v", err)
	}
	if meta.Duration == 0 || meta.Width == 0 || meta.Height == 0 {
		t.Fatalf("invalid video metadata: %+v", meta)
	}
}

func closeTestFile(t *testing.T, file *os.File) {
	t.Helper()
	if err := file.Close(); err != nil {
		t.Errorf("close test file: %v", err)
	}
}
