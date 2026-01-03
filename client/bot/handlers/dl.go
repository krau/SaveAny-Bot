package handlers

import (
	"net/url"
	"strings"
	"sync"

	"github.com/celestix/gotgproto/ext"
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
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, update.GetUserChat().GetID()), tcbdata.Add{
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

var aria2ClientInitOnce sync.Once
var aria2ClientInitErr error
var aria2Client *aria2.Client

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
	logger.Debug("Adding aria2 download", "links", links)
	aria2ClientInitOnce.Do(func() {
		aria2Client, aria2ClientInitErr = aria2.NewClient(config.C().Aria2.Url, config.C().Aria2.Secret)
	})
	if aria2ClientInitErr != nil {
		logger.Error("Failed to initialize aria2 client", "error", aria2ClientInitErr)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgAria2ErrorAria2ClientInitFailed, map[string]any{
			"Error": aria2ClientInitErr.Error(),
		})), nil)
		return nil
	}
	gid, err := aria2Client.AddURI(ctx, links, nil)
	if err != nil {
		logger.Error("Failed to add aria2 download", "error", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgAria2ErrorAddingAria2Download, map[string]any{
			"Error": err.Error(),
		})), nil)
		return nil
	}
	logger.Info("Aria2 download added", "gid", gid)
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgAria2InfoAria2DownloadAdded, map[string]any{
		"GID": gid,
	})), nil)
	return nil
}
