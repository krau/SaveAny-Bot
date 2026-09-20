package ytdlp

import (
	"strings"

	"github.com/krau/SaveAny-Bot/config"
)

// splitOutputTemplate pulls a user-supplied yt-dlp output template out of the
// custom flags: the last one wins, matching yt-dlp. A dangling -o/--output is
// kept so yt-dlp reports the missing value itself.
func splitOutputTemplate(flags []string) (template string, rest []string) {
	rest = make([]string, 0, len(flags))
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		if flag == "-o" || flag == "--output" {
			if i+1 == len(flags) {
				rest = append(rest, flag)
				continue
			}
			i++
			template = flags[i]
			continue
		}
		switch {
		case strings.HasPrefix(flag, "--output="):
			template = strings.TrimPrefix(flag, "--output=")
		case strings.HasPrefix(flag, "-o") && len(flag) > 2:
			template = flag[2:]
		default:
			rest = append(rest, flag)
		}
	}
	return template, rest
}

// resolveFilenameTemplate picks the output template: a user-provided
// -o/--output, then ytdlp.filename_template, then the built-in default.
func resolveFilenameTemplate(cfg config.YtdlpConfig, userTemplate string) string {
	switch {
	case userTemplate != "":
		return userTemplate
	case cfg.FilenameTemplate != "":
		return cfg.FilenameTemplate
	default:
		return config.DefaultYtdlpFilenameTemplate
	}
}
