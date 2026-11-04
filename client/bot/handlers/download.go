package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/aria2"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
	"github.com/rs/xid"
)

type downloadOption struct {
	label i18nk.Key
	data  tcbdata.Add
}

// handleDownloadCmd parses links once, then lets the user choose the download
// engine selection.
func handleDownloadCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := splitQuotedArgs(update.EffectiveMessage.Text)
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDownloadUsage)), nil)
		return dispatcher.EndGroups
	}

	webLinks, flags := parseYtdlpArgs(logger, args[1:])
	allLinks := parseDownloadURIs(args[1:], webLinks)
	if len(webLinks) == 0 && len(allLinks) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDownloadErrorNoValidLinks)), nil)
		return dispatcher.EndGroups
	}

	options := make([]downloadOption, 0, 3)
	if len(webLinks) > 0 && len(flags) == 0 {
		options = append(options, downloadOption{
			label: i18nk.BotMsgDownloadButtonDirect,
			data: tcbdata.Add{
				TaskType:    tasktype.TaskTypeDirectlinks,
				DirectLinks: webLinks,
			},
		})
	}
	if config.C().Aria2.Enable && len(allLinks) > 0 {
		options = append(options, downloadOption{
			label: i18nk.BotMsgDownloadButtonAria2,
			data: tcbdata.Add{
				TaskType:  tasktype.TaskTypeAria2,
				Aria2URIs: allLinks,
			},
		})
	}
	if len(webLinks) > 0 {
		options = append(options, downloadOption{
			label: i18nk.BotMsgDownloadButtonVideo,
			data: tcbdata.Add{
				TaskType:   tasktype.TaskTypeYtdlp,
				YtdlpURLs:  webLinks,
				YtdlpFlags: flags,
			},
		})
	}
	if len(options) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDownloadErrorNoEngine)), nil)
		return dispatcher.EndGroups
	}

	buttons := make([]tg.KeyboardInlineButton, 0, len(options))
	for _, option := range options {
		dataID := xid.New().String()
		if err := cache.Set(dataID, option.data); err != nil {
			return err
		}
		buttons = append(buttons, markup.Callback(
			i18n.T(option.label, nil),
			fmt.Appendf(nil, "%s %s", tcbdata.TypeEngine, dataID),
		))
	}
	markup := &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{{Buttons: buttons}}}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDownloadPromptSelectEngine, map[string]any{
		"Count": len(allLinks),
	})), &ext.ReplyOpts{Markup: markup})
	return dispatcher.EndGroups
}

func parseDownloadURIs(args, webLinks []string) []string {
	links := make([]string, 0, len(args))
	seen := make(map[string]struct{}, len(args))
	add := func(link string) {
		if link == "" {
			return
		}
		if _, ok := seen[link]; ok {
			return
		}
		seen[link] = struct{}{}
		links = append(links, link)
	}
	for _, link := range webLinks {
		add(link)
	}
	for _, arg := range args {
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		u, err := url.Parse(arg)
		if err == nil && u.Scheme != "" {
			add(arg)
		}
	}
	return links
}

func handleEngineCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	data, err := shortcut.GetCallbackDataWithAnswer[tcbdata.Add](ctx, update, args[1])
	if err != nil {
		return err
	}
	storages := storage.GetUserStorages(ctx, update.CallbackQuery.GetUserID())
	if len(storages) == 0 {
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), i18n.T(i18nk.BotMsgCommonErrorNoAvailableStorage)))
		return dispatcher.EndGroups
	}
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storages, data)
	if err != nil {
		return err
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     i18n.T(i18nk.BotMsgDownloadInfoSelectStorage, map[string]any{"Count": downloadOptionCount(data)}),
		ReplyMarkup: markup,
	})
	return dispatcher.EndGroups
}

func downloadOptionCount(data tcbdata.Add) int {
	switch data.TaskType {
	case tasktype.TaskTypeDirectlinks:
		return len(data.DirectLinks)
	case tasktype.TaskTypeAria2:
		return len(data.Aria2URIs)
	case tasktype.TaskTypeYtdlp:
		return len(data.YtdlpURLs)
	default:
		return 0
	}
}

var aria2ClientCache struct {
	sync.Mutex
	key    string
	client *aria2.Client
}

// GetAria2Client returns a client for the current aria2 settings. The cache is
// keyed by URL and secret so changing settings does not require a restart.
func GetAria2Client() (*aria2.Client, error) {
	cfg := config.C().Aria2
	key := cfg.Url + "\x00" + cfg.Secret
	aria2ClientCache.Lock()
	defer aria2ClientCache.Unlock()
	if aria2ClientCache.client != nil && aria2ClientCache.key == key {
		return aria2ClientCache.client, nil
	}
	client, err := aria2.NewClient(cfg.Url, cfg.Secret)
	if err != nil {
		return nil, err
	}
	aria2ClientCache.client = client
	aria2ClientCache.key = key
	return client, nil
}
