package ytdlp

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
)

// TestDefaultFilenameTemplateLongTitle reproduces "[Errno 36] Filename too long" for
// posts whose caption becomes the title. It needs a local yt-dlp and only
// renders the template from an info JSON, so it never touches the network.
func TestDefaultFilenameTemplateLongTitle(t *testing.T) {
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		t.Skip("yt-dlp not installed")
	}

	cases := []struct {
		name  string
		title string
	}{
		{name: "ascii", title: strings.Repeat("Cita-cita_terbesar_kita_", 20)},
		{name: "multibyte", title: strings.Repeat("آية الكرسي ", 40)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			info := filepath.Join(dir, "info.json")
			writeInfoJSON(t, info, tc.title)

			out, err := exec.Command(bin, "--ignore-config", "--load-info-json", info,
				"-o", filepath.Join(dir, config.DefaultYtdlpFilenameTemplate), "--print", "filename").Output()
			if err != nil {
				t.Fatalf("yt-dlp: %v", err)
			}
			name := filepath.Base(strings.TrimSpace(string(out)))
			// ".part" and format ids (".f140") are appended later; keep room for them.
			if got := len(name + ".f140.part"); got > 255 {
				t.Fatalf("filename is %d bytes, want <= 255: %q", got, name)
			}
			if !strings.HasSuffix(name, ".m4a") {
				t.Fatalf("extension lost: %q", name)
			}
		})
	}
}

func writeInfoJSON(t *testing.T, path, title string) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"id":            "abc123",
		"title":         title,
		"ext":           "m4a",
		"extractor":     "generic",
		"extractor_key": "Generic",
		"webpage_url":   "https://example.invalid/post",
		"url":           "https://example.invalid/media.m4a",
		"formats":       []map[string]any{{"format_id": "0", "url": "https://example.invalid/media.m4a", "ext": "m4a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}
