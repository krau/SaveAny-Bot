package config

import (
	"fmt"
	"net/url"
	"strings"
)

// DefaultYtdlpFilenameTemplate is used when ytdlp.filename_template is not set.
// The title is capped at 200 bytes so names stay under the 255-byte NAME_MAX
// limit, with room for suffixes such as ".f140.m4a.part".
const DefaultYtdlpFilenameTemplate = "%(title).200B.%(ext)s"

// LegacyYtdlpFilenameTemplate is the uncapped default still stored by configs
// created before the title cap; it is treated as unset when settings load.
const LegacyYtdlpFilenameTemplate = "%(title)s.%(ext)s"

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
// Blank entries are ignored. Errors never echo the entry, since it may contain
// a password.
func (c YtdlpConfig) ValidateProxies() error {
	for i, raw := range c.Proxies {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid yt-dlp proxy %d: malformed URL", i+1)
		}
		switch u.Scheme {
		case "http", "https", "socks4", "socks4a", "socks5", "socks5h":
		default:
			return fmt.Errorf("invalid yt-dlp proxy %d: scheme %q must be http, https, socks4, socks4a, socks5 or socks5h", i+1, u.Scheme)
		}
		if u.Host == "" {
			return fmt.Errorf("invalid yt-dlp proxy %d: missing host", i+1)
		}
	}
	return nil
}
