package handlers

import (
	"net/url"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/aria2"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/ext"
)

func handleDlCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDlUsage)), nil)
		return nil
	}
	links := args[1:]
	for i, link := range links {
		links[i] = strings.TrimSpace(link)
		u, err := url.Parse(link)
		if err != nil || u.Scheme == "" || u.Host == "" {
			logger.Warn("invaild link", link)
			links[i] = ""
		}
	}
	links = slice.Compact(links)
	if len(links) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDlErrorNoValidLinks)), nil)
		return nil
	}
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, responsibleUserID(update)), tcbdata.Add{
		TaskType:    tasktype.TaskTypeDirectlinks,
		DirectLinks: links,
	})
	if err != nil {
		return err
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDlInfoFilesSelectStorage, map[string]any{
		"Count": len(links),
	})), &ext.ReplyOpts{
		Markup: markup,
	})
	return nil
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

func handleAria2DlCmd(ctx *ext.Context, update *ext.Update) error {
	if !config.C().Aria2.Enable {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgAria2ErrorAria2NotEnabled)), nil)
		return nil
	}
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDlUsage)), nil)
		return nil
	}
	links := args[1:]
	for i, link := range links {
		links[i] = strings.TrimSpace(link)
	}
	links = slice.Compact(links)
	if len(links) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDlErrorNoValidLinks)), nil)
		return nil
	}
	logger.Debug("Preparing aria2 download", "links", links)

	// Initialize aria2 client to check connection.
	if _, err := GetAria2Client(); err != nil {
		logger.Error("Failed to initialize aria2 client", "error", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgAria2ErrorAria2ClientInitFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return nil
	}

	// don't add to aria2 yet
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, responsibleUserID(update)), tcbdata.Add{
		TaskType:  tasktype.TaskTypeAria2,
		Aria2URIs: links,
	})
	if err != nil {
		return err
	}

	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgAria2InfoSelectStorage)), &ext.ReplyOpts{
		Markup: markup,
	})
	return nil
}
