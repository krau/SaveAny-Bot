package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestYtdlpEnvBinding guards that every ytdlp option is registered as a viper
// default, which is what lets SAVEANY_ loads reach it.
func TestYtdlpEnvBinding(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SAVEANY_YTDLP_RESTRICT_FILENAMES", "true")
	t.Setenv("SAVEANY_YTDLP_FILENAME_TEMPLATE", "%(id)s.%(ext)s")
	t.Setenv("SAVEANY_YTDLP_MAX_HEIGHT", "720")
	t.Setenv("SAVEANY_YTDLP_FORMAT", "best")
	t.Setenv("SAVEANY_YTDLP_RECODE", "mkv")

	if err := Init(t.Context(), configPath); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	got := C().Ytdlp
	if !got.RestrictFilenames {
		t.Error("restrict_filenames was not read from the environment")
	}
	if got.FilenameTemplate != "%(id)s.%(ext)s" {
		t.Errorf("filename_template = %q, want %q", got.FilenameTemplate, "%(id)s.%(ext)s")
	}
	if got.MaxHeight != 720 {
		t.Errorf("max_height = %d, want 720", got.MaxHeight)
	}
	if got.Format != "best" {
		t.Errorf("format = %q, want %q", got.Format, "best")
	}
	if got.Recode != "mkv" {
		t.Errorf("recode = %q, want %q", got.Recode, "mkv")
	}
}
