package handlers

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"

	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleYtdlpCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := splitQuotedArgs(update.EffectiveMessage.Text)
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgYtdlpUsage)), nil)
		return dispatcher.EndGroups
	}

	urls, flags := parseYtdlpArgs(logger, args[1:])

	if len(urls) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgYtdlpErrorNoValidUrls)), nil)
		return dispatcher.EndGroups
	}

	logger.Debugf("Preparing yt-dlp download for %d URL(s) with %d flag(s)", len(urls), len(flags))

	// Build storage selection keyboard
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, update.GetUserChat().GetID()), tcbdata.Add{
		TaskType:   tasktype.TaskTypeYtdlp,
		YtdlpURLs:  urls,
		YtdlpFlags: flags,
	})
	if err != nil {
		return err
	}

	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgYtdlpInfoUrlsSelectStorage, map[string]any{
		"Count": len(urls),
	})), &ext.ReplyOpts{
		Markup: markup,
	})

	return dispatcher.EndGroups
}

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

// parseYtdlpArgs separates the /ytdlp arguments into URLs and custom yt-dlp
// flags. A flag value is kept together with its flag, unless it looks like a URL
// of its own.
func parseYtdlpArgs(logger *log.Logger, args []string) (urls []string, flags []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}

		// Check if it's a flag (starts with - or --)
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if the next argument might be a value for this flag
			// Don't consume it if it starts with - or looks like a URL with scheme
			if i+1 < len(args) {
				nextArg := args[i+1]
				if nextArg != "" && !strings.HasPrefix(nextArg, "-") {
					// Check if it's clearly a URL (has ://)
					// This handles common video URLs (http://, https://)
					// For other yt-dlp inputs, users should ensure proper formatting
					if strings.Contains(nextArg, "://") {
						// It's a URL, don't consume it as a flag value
						continue
					}
					// Otherwise, treat it as a flag value
					flags = append(flags, nextArg)
					i++ // Skip the next argument as it's been consumed
				}
			}
		} else {
			// Try to parse as URL
			u, err := url.Parse(arg)
			if err != nil || u.Scheme == "" || u.Host == "" {
				logger.Warnf("Invalid URL: %s", arg)
				continue
			}
			urls = append(urls, arg)
		}
	}
	return urls, flags
}
