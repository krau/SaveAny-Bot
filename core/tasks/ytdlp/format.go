package ytdlp

import (
	"strconv"

	ytdlp "github.com/lrstanley/go-ytdlp"

	"github.com/krau/SaveAny-Bot/config"
)

// buildFormatSelector translates a max height into a yt-dlp format selector.
// It prefers merging the best video+audio within the height limit, then falls
// back to a single muxed stream. An empty result means "no explicit selector".
func buildFormatSelector(maxHeight int) string {
	if maxHeight <= 0 {
		return ""
	}
	h := strconv.Itoa(maxHeight)
	return "bv*[height<=" + h + "]+ba/b[height<=" + h + "]/b"
}

// applyFormatConfig configures format/quality on the yt-dlp command according to
// the ytdlp config. It is only meant to be called when the user did not supply
// any custom flags, so config-driven defaults never conflict with user input.
func applyFormatConfig(cmd *ytdlp.Command, cfg config.YtdlpConfig) *ytdlp.Command {
	switch {
	case cfg.Format != "":
		cmd = cmd.Format(cfg.Format)
	case cfg.MaxHeight > 0:
		cmd = cmd.Format(buildFormatSelector(cfg.MaxHeight))
	default:
		// Preserve the original default: prefer highest resolution mp4/m4a.
		cmd = cmd.FormatSort("res,ext:mp4:m4a")
	}
	if cfg.Recode != "" {
		cmd = cmd.RecodeVideo(cfg.Recode)
	}
	cmd = cmd.RestrictFilenames()
	return cmd
}
