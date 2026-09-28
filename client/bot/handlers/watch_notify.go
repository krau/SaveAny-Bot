package handlers

import (
	"context"
	"strings"
	"sync"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"

	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
)

// handleWatchNotifyCmd toggles the task notifications for watched chats and
// reports the current state, which is disabled by default.
func handleWatchNotifyCmd(ctx *ext.Context, update *ext.Update, user *database.User, args []string) error {
	if len(args) > 0 {
		enabled, ok := parseWatchNotifyArg(args[0])
		if !ok {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchErrorNotifyArgInvalid)), nil)
			return dispatcher.EndGroups
		}
		user.WatchNotify = enabled
		if err := database.UpdateUser(ctx, user); err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorUpdateUserInfoFailed, map[string]any{
				"Error": err.Error(),
			})), nil)
			return dispatcher.EndGroups
		}
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(watchNotifyStateKey(user.WatchNotify))), nil)
	return dispatcher.EndGroups
}

// parseWatchNotifyArg parses the argument of /watch notify.
func parseWatchNotifyArg(arg string) (enabled bool, ok bool) {
	switch strings.ToLower(arg) {
	case "on":
		return true, true
	case "off":
		return false, true
	default:
		return false, false
	}
}

func watchNotifyStateKey(enabled bool) i18nk.Key {
	if enabled {
		return i18nk.BotMsgWatchInfoNotifyEnabled
	}
	return i18nk.BotMsgWatchInfoNotifyDisabled
}

// openWatchNotification sends the notification for a watched task and returns the
// tracker that keeps editing it, or nil when the message could not be sent.
var openWatchNotification = func(ctx context.Context, botCtx *ext.Context, chatID int64, fileName string) tftask.ProgressTracker {
	text, entities, err := watchNotifyMarkup(fileName)
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to render watch notification: %s", err)
		return nil
	}
	msg, err := botCtx.SendMessage(chatID, &tg.MessagesSendMessageRequest{
		Message:  text,
		Entities: entities,
	})
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to send watch notification to user %d: %s", chatID, err)
		return nil
	}
	return tftask.NewProgressTrack(msg.ID, chatID)
}

// watchNotifyProgress reports a watched task through a notification the bot sends
// on first use, so sending never blocks the watched-chat listener. The bot context
// is a per-task copy: watched tasks are created and run on several goroutines, and
// an ext context keeps a message ID random source that is not safe for concurrent
// use.
type watchNotifyProgress struct {
	botCtx *ext.Context
	chatID int64
	once   sync.Once
	inner  tftask.ProgressTracker
}

// newWatchNotifyProgress returns the tracker reporting a watched task to the user,
// or nil when notifications are disabled.
func newWatchNotifyProgress(ctx context.Context, botCtx *ext.Context, user *database.User) tftask.ProgressTracker {
	if !user.WatchNotify || botCtx == nil {
		return nil
	}
	return &watchNotifyProgress{botCtx: newBotContext(ctx, botCtx), chatID: user.ChatID}
}

// newBotContext copies the bot context for a single task notification. Entities
// are left out on purpose: sending and editing messages does not need them, and
// the shared map would be one more thing to share across goroutines.
func newBotContext(ctx context.Context, botCtx *ext.Context) *ext.Context {
	return ext.NewContext(ctx, botCtx.Raw, botCtx.PeerStorage, botCtx.Self, botCtx.Sender, nil, false)
}

// progress returns the tracker of the notification, opening it on first use.
func (p *watchNotifyProgress) progress(ctx context.Context, info tftask.TaskInfo) tftask.ProgressTracker {
	p.once.Do(func() {
		p.inner = openWatchNotification(ctx, p.botCtx, p.chatID, info.FileName())
	})
	return p.inner
}

// botCtxFor carries the bot context the notification is reported through: the task
// context keeps the userbot, which downloads the media and uploads to the Telegram
// storage backend, so it must stay in place.
func (p *watchNotifyProgress) botCtxFor(ctx context.Context) context.Context {
	return tgutil.ExtWithContext(ctx, p.botCtx)
}

func (p *watchNotifyProgress) OnStart(ctx context.Context, info tftask.TaskInfo) {
	if tracker := p.progress(ctx, info); tracker != nil {
		tracker.OnStart(p.botCtxFor(ctx), info)
	}
}

func (p *watchNotifyProgress) OnProgress(ctx context.Context, info tftask.TaskInfo, downloaded, total int64) {
	if tracker := p.progress(ctx, info); tracker != nil {
		tracker.OnProgress(p.botCtxFor(ctx), info, downloaded, total)
	}
}

func (p *watchNotifyProgress) OnUploadStart(ctx context.Context, info tftask.TaskInfo, total int64) {
	if tracker := p.progress(ctx, info); tracker != nil {
		if upload, ok := tracker.(tftask.UploadProgressTracker); ok {
			upload.OnUploadStart(p.botCtxFor(ctx), info, total)
		}
	}
}

func (p *watchNotifyProgress) OnUploadProgress(ctx context.Context, info tftask.TaskInfo, uploaded, total int64) {
	if tracker := p.progress(ctx, info); tracker != nil {
		if upload, ok := tracker.(tftask.UploadProgressTracker); ok {
			upload.OnUploadProgress(p.botCtxFor(ctx), info, uploaded, total)
		}
	}
}

func (p *watchNotifyProgress) OnDone(ctx context.Context, info tftask.TaskInfo, err error) {
	if tracker := p.progress(ctx, info); tracker != nil {
		tracker.OnDone(p.botCtxFor(ctx), info, err)
	}
}

// watchNotifyMarkup renders the notification shown for a detected media message.
func watchNotifyMarkup(fileName string) (string, []tg.MessageEntityClass, error) {
	return tgutil.RenderHTML(i18n.T(i18nk.BotMsgWatchNotifyDetected, tgutil.EscapeHTMLTemplateData(map[string]any{
		"Name": fileName,
	})))
}
