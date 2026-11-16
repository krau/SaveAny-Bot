package config

import (
	"fmt"
	"net/url"
	"strings"
)

// DefaultYtdlpFilenameTemplate is used when ytdlp.filename_template is not set.
const DefaultYtdlpFilenameTemplate = "%(title)s.%(ext)s"

type YtdlpConfig struct {
	// MaxHeight limits the video resolution by height in pixels (e.g. 1080, 720).
	// 0 means no limit (best available). Ignored when Format is set.
	MaxHeight int `toml:"max_height" mapstructure:"max_height" json:"max_height"`
	// Format is a raw yt-dlp format selector (-f). When set, it takes precedence
	// over MaxHeight and gives the user full control.
	Format string `toml:"format" mapstructure:"format" json:"format"`
	// Recode is the target video container yt-dlp recodes into (e.g. mp4).
	// Empty disables recoding.
	Recode string `toml:"recode" mapstructure:"recode" json:"recode"`
	// FilenameTemplate is the yt-dlp output template for downloaded files, e.g.
	// "%(uploader)s - %(title)s.%(ext)s", relative to the bot's temp directory.
	FilenameTemplate string `toml:"filename_template" mapstructure:"filename_template" json:"filename_template"`
	// RestrictFilenames keeps file names ASCII-only (yt-dlp --restrict-filenames),
	// dropping non-latin characters from titles. Disabled by default.
	RestrictFilenames bool `toml:"restrict_filenames" mapstructure:"restrict_filenames" json:"restrict_filenames"`
	// Proxies are HTTP/HTTPS/SOCKS proxy URLs for yt-dlp's own requests. They
	// are tried in order until one succeeds, so put the preferred proxy first.
	// Empty means yt-dlp connects directly.
	Proxies []string `toml:"proxies" mapstructure:"proxies" json:"proxies"`
}

// ValidateProxies reports the first entry in Proxies that yt-dlp cannot use.
// Blank entries are ignored.
func (c YtdlpConfig) ValidateProxies() error {
	for _, raw := range c.Proxies {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid yt-dlp proxy: %w", err)
		}
		switch u.Scheme {
		case "http", "https", "socks4", "socks4a", "socks5", "socks5h":
		default:
			return fmt.Errorf("invalid yt-dlp proxy %q: scheme must be http, https, socks4, socks4a, socks5 or socks5h", u.Redacted())
		}
		if u.Host == "" {
			return fmt.Errorf("invalid yt-dlp proxy %q: missing host", u.Redacted())
		}
	}
	return nil
}
