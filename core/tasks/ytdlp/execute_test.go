package ytdlp

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	ytdlp "github.com/lrstanley/go-ytdlp"

	"github.com/krau/SaveAny-Bot/config"
)

// commandArgs renders the flags the library would pass to yt-dlp.
func commandArgs(cmd *ytdlp.Command) []string {
	var args []string
	for _, flag := range cmd.GetFlagConfig().ToFlags() {
		args = append(args, flag.Raw()...)
	}
	return args
}

func TestBuildDownloadCommand(t *testing.T) {
	const tempDir = "/tmp/ytdlp-test"
	tests := []struct {
		name            string
		cfg             config.YtdlpConfig
		flags           []string
		wantTemplate    string
		wantFlags       []string
		wantRestricted  bool
		wantFormatFlags bool
	}{
		{
			name:            "default template keeps titles",
			wantTemplate:    "%(title)s.%(ext)s",
			wantFormatFlags: true,
		},
		{
			name:            "config template",
			cfg:             config.YtdlpConfig{FilenameTemplate: "%(uploader)s - %(title)s.%(ext)s"},
			wantTemplate:    "%(uploader)s - %(title)s.%(ext)s",
			wantFormatFlags: true,
		},
		{
			name:            "user template overrides config template",
			cfg:             config.YtdlpConfig{FilenameTemplate: "%(id)s.%(ext)s"},
			flags:           []string{"-o", "%(title)s [%(id)s].%(ext)s"},
			wantTemplate:    "%(title)s [%(id)s].%(ext)s",
			wantFormatFlags: true,
		},
		{
			name:            "restrict filenames only when configured",
			cfg:             config.YtdlpConfig{RestrictFilenames: true, MaxHeight: 1080},
			wantTemplate:    "%(title)s.%(ext)s",
			wantRestricted:  true,
			wantFormatFlags: true,
		},
		{
			name:            "custom flags skip format defaults",
			cfg:             config.YtdlpConfig{MaxHeight: 1080},
			flags:           []string{"-f", "best"},
			wantTemplate:    "%(title)s.%(ext)s",
			wantFlags:       []string{"-f", "best"},
			wantFormatFlags: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, flags, err := buildDownloadCommand(tt.cfg, tempDir, tt.flags)
			if err != nil {
				t.Fatalf("buildDownloadCommand() failed: %v", err)
			}
			args := commandArgs(cmd)

			idx := slices.Index(args, "--output")
			if idx < 0 || idx+1 >= len(args) {
				t.Fatalf("--output not found in %q", args)
			}
			if got, want := args[idx+1], filepath.Join(tempDir, tt.wantTemplate); got != want {
				t.Errorf("output path = %q, want %q", got, want)
			}
			if got := slices.Contains(args, "--restrict-filenames"); got != tt.wantRestricted {
				t.Errorf("--restrict-filenames = %v, want %v (args: %q)", got, tt.wantRestricted, args)
			}
			formatFlags := slices.Contains(args, "--format") || slices.Contains(args, "--format-sort") || slices.Contains(args, "--recode-video")
			if formatFlags != tt.wantFormatFlags {
				t.Errorf("format defaults applied = %v, want %v (args: %q)", formatFlags, tt.wantFormatFlags, args)
			}
			if !slices.Equal(flags, tt.wantFlags) {
				t.Errorf("flags = %q, want %q", flags, tt.wantFlags)
			}
		})
	}
}

func TestCollectDownloadedFiles(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"video.mp4", filepath.Join("playlist", "ep1.mp4"), filepath.Join("playlist", "ep2.mp4")} {
		path := filepath.Join(tempDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectDownloadedFiles(tempDir)
	if err != nil {
		t.Fatalf("collectDownloadedFiles() error = %v", err)
	}

	want := []string{
		filepath.Join(tempDir, "playlist", "ep1.mp4"),
		filepath.Join(tempDir, "playlist", "ep2.mp4"),
		filepath.Join(tempDir, "video.mp4"),
	}
	if !slices.Equal(files, want) {
		t.Errorf("collectDownloadedFiles() = %q, want %q", files, want)
	}
}
