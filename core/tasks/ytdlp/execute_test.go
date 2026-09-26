package ytdlp

import (
	"context"
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

// outputArgs lists the output templates of an argv, in order.
func outputArgs(args []string) []string {
	var outputs []string
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--output" || args[i] == "-o" {
			outputs = append(outputs, args[i+1])
		}
	}
	return outputs
}

func TestBuildDownloadCommand(t *testing.T) {
	const tempDir = "/tmp/ytdlp-test"
	tests := []struct {
		name            string
		cfg             config.YtdlpConfig
		flags           []string
		wantOutputs     []string
		wantFlags       []string
		wantRestricted  bool
		wantFormatFlags bool
	}{
		{
			name:            "default template roots the output in the temp dir",
			wantOutputs:     []string{filepath.Join(tempDir, "%(title)s.%(ext)s")},
			wantFormatFlags: true,
		},
		{
			name:            "config template roots the output in the temp dir",
			cfg:             config.YtdlpConfig{FilenameTemplate: "%(uploader)s - %(title)s.%(ext)s"},
			wantOutputs:     []string{filepath.Join(tempDir, "%(uploader)s - %(title)s.%(ext)s")},
			wantFormatFlags: true,
		},
		{
			name:  "user template comes after the base template",
			cfg:   config.YtdlpConfig{FilenameTemplate: "%(id)s.%(ext)s", MaxHeight: 1080},
			flags: []string{"-o", "%(title)s [%(id)s].%(ext)s"},
			wantOutputs: []string{
				filepath.Join(tempDir, "%(id)s.%(ext)s"),
				filepath.Join(tempDir, "%(title)s [%(id)s].%(ext)s"),
			},
			wantFlags:       []string{"-o", filepath.Join(tempDir, "%(title)s [%(id)s].%(ext)s")},
			wantFormatFlags: false,
		},
		{
			name:  "prefixed template keeps the base template for other outputs",
			flags: []string{"-o", "subtitle:subs/%(title)s.%(ext)s"},
			wantOutputs: []string{
				filepath.Join(tempDir, "%(title)s.%(ext)s"),
				"subtitle:" + filepath.Join(tempDir, "subs", "%(title)s.%(ext)s"),
			},
			wantFlags:       []string{"-o", "subtitle:" + filepath.Join(tempDir, "subs", "%(title)s.%(ext)s")},
			wantFormatFlags: false,
		},
		{
			name:            "restrict filenames only when configured",
			cfg:             config.YtdlpConfig{RestrictFilenames: true, MaxHeight: 1080},
			wantOutputs:     []string{filepath.Join(tempDir, "%(title)s.%(ext)s")},
			wantRestricted:  true,
			wantFormatFlags: true,
		},
		{
			name:            "unrelated flags skip format defaults",
			cfg:             config.YtdlpConfig{MaxHeight: 1080},
			flags:           []string{"-f", "best"},
			wantOutputs:     []string{filepath.Join(tempDir, "%(title)s.%(ext)s")},
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
			args := append(commandArgs(cmd), flags...)

			if got := outputArgs(args); !slices.Equal(got, tt.wantOutputs) {
				t.Errorf("outputs = %q, want %q (args: %q)", got, tt.wantOutputs, args)
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

func TestBuildDownloadCommandRejectsBadTemplates(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
	}{
		{name: "escaping template", flags: []string{"-o", "../escaped/%(title)s.%(ext)s"}},
		{name: "dangling flag", flags: []string{"-o"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if cmd, flags, err := buildDownloadCommand(config.YtdlpConfig{}, "/tmp/ytdlp-test", tt.flags); err == nil {
				t.Fatalf("buildDownloadCommand(%q) = (%v, %q), want an error", tt.flags, cmd, flags)
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
		filepath.Join("playlist", "ep1.mp4"),
		filepath.Join("playlist", "ep2.mp4"),
		"video.mp4",
	}
	if !slices.Equal(files, want) {
		t.Errorf("collectDownloadedFiles() = %q, want %q", files, want)
	}
}

func TestTransferFileKeepsTemplateDirectories(t *testing.T) {
	tempDir := t.TempDir()
	relPath := filepath.Join("sub", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(tempDir, relPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, relPath), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	stor := &MockStorage{}
	progress := &recordingProgress{}
	task := NewTask("transfer", t.Context(), []string{"https://example.com/video"}, nil, stor, "videos", progress)

	if err := task.transferFile(t.Context(), tempDir, relPath); err != nil {
		t.Fatalf("transferFile() failed: %v", err)
	}

	want := filepath.Join("videos", "sub", "video.mp4")
	if !slices.Equal(stor.saved, []string{want}) {
		t.Errorf("saved files = %q, want %q", stor.saved, []string{want})
	}
	if wantStatus := "Transferred: " + relPath; !slices.Contains(progress.statuses, wantStatus) {
		t.Errorf("progress statuses = %q, want %q", progress.statuses, wantStatus)
	}
}

// recordingProgress records the statuses reported for a task.
type recordingProgress struct {
	statuses []string
}

func (p *recordingProgress) OnStart(context.Context, *Task) {}

func (p *recordingProgress) OnProgress(_ context.Context, _ *Task, status string) {
	p.statuses = append(p.statuses, status)
}

func (p *recordingProgress) OnDone(context.Context, *Task, error) {}
