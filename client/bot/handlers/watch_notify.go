package handlers

import (
	"context"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"

	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
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

// watchNotifyProgress reports the progress of a task created by a watched chat
// through the standard file progress tracker, with the bot context injected: the
// task context carries the userbot, which downloads the media and uploads to the
// Telegram storage backend, so it must stay in place.
type watchNotifyProgress struct {
	ext     *ext.Context
	tracker tftask.ProgressTracker
}

func (p watchNotifyProgress) botCtx(ctx context.Context) context.Context {
	return tgutil.ExtWithContext(ctx, p.ext)
}

func (p watchNotifyProgress) OnStart(ctx context.Context, info tftask.TaskInfo) {
	p.tracker.OnStart(p.botCtx(ctx), info)
}

func (p watchNotifyProgress) OnProgress(ctx context.Context, info tftask.TaskInfo, downloaded, total int64) {
	p.tracker.OnProgress(p.botCtx(ctx), info, downloaded, total)
}

func (p watchNotifyProgress) OnUploadStart(ctx context.Context, info tftask.TaskInfo, total int64) {
	if upload, ok := p.tracker.(tftask.UploadProgressTracker); ok {
		upload.OnUploadStart(p.botCtx(ctx), info, total)
	}
}

func (p watchNotifyProgress) OnUploadProgress(ctx context.Context, info tftask.TaskInfo, uploaded, total int64) {
	if upload, ok := p.tracker.(tftask.UploadProgressTracker); ok {
		upload.OnUploadProgress(p.botCtx(ctx), info, uploaded, total)
	}
}

func (p watchNotifyProgress) OnDone(ctx context.Context, info tftask.TaskInfo, err error) {
	p.tracker.OnDone(p.botCtx(ctx), info, err)
}

// newWatchNotifyProgress notifies the user about the media message picked up by a
// watched chat and returns the tracker that keeps updating the notification with
// the task result. It returns nil when notifications are disabled or the message
// could not be sent, leaving the task without progress reporting.
func newWatchNotifyProgress(ctx context.Context, botCtx *ext.Context, user *database.User, file tfile.TGFileMessage) tftask.ProgressTracker {
	if !user.WatchNotify || botCtx == nil {
		return nil
	}
	logger := log.FromContext(ctx)
	text, entities, err := watchNotifyMarkup(file.Name())
	if err != nil {
		logger.Errorf("Failed to render watch notification: %s", err)
		return nil
	}
	msg, err := botCtx.SendMessage(user.ChatID, &tg.MessagesSendMessageRequest{
		Message:  text,
		Entities: entities,
	})
	if err != nil {
		logger.Errorf("Failed to send watch notification to user %d: %s", user.ChatID, err)
		return nil
	}
	return watchNotifyProgress{ext: botCtx, tracker: tftask.NewProgressTrack(msg.ID, user.ChatID)}
}

// watchNotifyMarkup renders the notification shown for a detected media message.
func watchNotifyMarkup(fileName string) (string, []tg.MessageEntityClass, error) {
	return tgutil.RenderHTML(i18n.T(i18nk.BotMsgWatchNotifyDetected, tgutil.EscapeHTMLTemplateData(map[string]any{
		"Name": fileName,
	})))
}
