package handlers

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/charmbracelet/log"
)

// splitQuotedArgs splits a command line on whitespace, keeping quoted values
// together and removing the quotes. Quotes only delimit a value when they start
// one, so apostrophes inside arguments stay literal.
func splitQuotedArgs(text string) []string {
	var args []string
	var current strings.Builder
	var quote rune
	for _, r := range text {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case (r == '"' || r == '\'') && current.Len() == 0:
			quote = r
		case unicode.IsSpace(r):
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// parseYtdlpArgs separates download arguments into URLs and custom yt-dlp
// flags. A flag value is kept together with its flag, unless it looks like a URL
// of its own.
func parseYtdlpArgs(logger *log.Logger, args []string) (urls []string, flags []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}

		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if i+1 < len(args) {
				nextArg := args[i+1]
				if nextArg != "" && !strings.HasPrefix(nextArg, "-") {
					// This handles common video URLs (http://, https://)
					// For other yt-dlp inputs, users should ensure proper formatting
					if strings.Contains(nextArg, "://") {
						continue
					}
					flags = append(flags, nextArg)
					i++ // Skip the next argument as it's been consumed
				}
			}
		} else {
			u, err := url.Parse(arg)
			if err != nil || u.Scheme == "" {
				logger.Warnf("Invalid URL: %s", arg)
				continue
			}
			// Magnet and other non-web schemes are valid Aria2 inputs but not
			// yt-dlp URLs; parseDownloadURIs handles them separately.
			if u.Host == "" {
				continue
			}
			urls = append(urls, arg)
		}
	}
	return urls, flags
}
