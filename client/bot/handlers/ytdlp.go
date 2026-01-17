package handlers

import (
	"net/url"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"

	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleYtdlpCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgYtdlpUsage)), nil)
		return dispatcher.EndGroups
	}

	urls := args[1:]
	// Validate and clean URLs
	for i, link := range urls {
		urls[i] = strings.TrimSpace(link)
		u, err := url.Parse(link)
		if err != nil || u.Scheme == "" || u.Host == "" {
			logger.Warnf("Invalid URL: %s", link)
			urls[i] = ""
		}
	}
	urls = slice.Compact(urls)

	if len(urls) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgYtdlpErrorNoValidUrls)), nil)
		return dispatcher.EndGroups
	}

	logger.Debugf("Preparing yt-dlp download for %d URL(s)", len(urls))

	// Build storage selection keyboard
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, update.GetUserChat().GetID()), tcbdata.Add{
		TaskType:  tasktype.TaskTypeYtdlp,
		YtdlpURLs: urls,
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
