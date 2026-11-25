package ytdlp

import (
	"strings"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
)

func TestNewCommandAppliesExecutableAndCookies(t *testing.T) {
	cfg := config.YtdlpConfig{
		Path:        " /opt/tools/yt-dlp ",
		CookiesFile: " /run/secrets/cookies.txt ",
	}
	cmd := NewCommand(cfg).BuildCommand(t.Context(), "https://example.com/video")

	if len(cmd.Args) == 0 || cmd.Args[0] != "/opt/tools/yt-dlp" {
		t.Fatalf("executable args = %#v", cmd.Args)
	}
	flags := strings.Join(cmd.Args, " ")
	if !strings.Contains(flags, "--cookies /run/secrets/cookies.txt") {
		t.Fatalf("args do not contain cookies option: %q", flags)
	}
}

func TestNewCommandLeavesOptionalSettingsUnset(t *testing.T) {
	cmd := NewCommand(config.YtdlpConfig{Path: "  ", CookiesFile: "  "}).
		BuildCommand(t.Context(), "https://example.com/video")

	if strings.Contains(strings.Join(cmd.Args, " "), "--cookies") {
		t.Fatalf("unexpected cookies option: %#v", cmd.Args)
	}
}
