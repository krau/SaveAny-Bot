package ytdlp

import (
	"strings"

	ytdlp "github.com/lrstanley/go-ytdlp"

	"github.com/krau/SaveAny-Bot/config"
)

// NewCommand creates a yt-dlp command with executable and authentication
// settings that must also apply to metadata lookups.
func NewCommand(cfg config.YtdlpConfig) *ytdlp.Command {
	cmd := ytdlp.New()
	if path := strings.TrimSpace(cfg.Path); path != "" {
		cmd = cmd.SetExecutable(path)
	}
	if cookies := strings.TrimSpace(cfg.CookiesFile); cookies != "" {
		cmd = cmd.Cookies(cookies)
	}
	return cmd
}
