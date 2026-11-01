package handlers

import (
	"errors"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

// responsibleUserID returns the sender's ID. Callback queries carry it
// natively; message updates resolve the sender through the entity map, which
// also covers group and supergroup chats.
func responsibleUserID(u *ext.Update) int64 {
	return tgutil.SenderID(u)
}

// chatID returns the chat that replies and edits must target. In private chats
// this equals the sender ID; in groups and supergroups it is the group chat.
func chatID(u *ext.Update) int64 {
	return tgutil.ChatID(u)
}

// isGroupChat reports whether the update belongs to a basic group or a
// supergroup. Broadcast channels are not group chats.
func isGroupChat(u *ext.Update) bool {
	if u == nil {
		return false
	}
	if message := u.EffectiveMessage; message != nil {
		switch message.PeerID.(type) {
		case *tg.PeerUser:
			return false
		case *tg.PeerChat:
			return true
		case *tg.PeerChannel:
			channel := u.GetChannel()
			return channel == nil || channel.Megagroup
		}
	}
	if u.GetChat() != nil {
		return true
	}
	channel := u.GetChannel()
	return channel != nil && channel.Megagroup
}

func checkPermission(ctx *ext.Context, update *ext.Update) error {
	userID := responsibleUserID(update)
	if !slice.Contain(config.C().GetUsersID(), userID) {
		if cbq := update.CallbackQuery; cbq != nil {
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(cbq.GetQueryID(), i18n.T(i18nk.BotMsgCommonErrorNoPermission, nil)))
		} else if !isGroupChat(update) {
			// Stay silent in groups: unaddressed traffic and unauthorized
			// command attempts must not turn into reply spam.
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorNoPermission, nil)), nil)
		}
		return dispatcher.EndGroups
	}

	return dispatcher.ContinueGroups
}

// withPermission wraps a callback handler with the same whitelist check used
// for message handlers (checkPermission). ContinueGroups is the dispatcher's
// success sentinel, not an error: only real failures and EndGroups stop the
// chain before the wrapped handler runs.
func withPermission(handler func(*ext.Context, *ext.Update) error) func(*ext.Context, *ext.Update) error {
	return func(ctx *ext.Context, update *ext.Update) error {
		if err := checkPermission(ctx, update); err != nil && !errors.Is(err, dispatcher.ContinueGroups) {
			return err
		}
		return handler(ctx, update)
	}
}

// skipLinkForMedia lets the media handler save an attached file instead of a
// link that appears in its caption.
func skipLinkForMedia(next func(*ext.Context, *ext.Update) error) func(*ext.Context, *ext.Update) error {
	return func(ctx *ext.Context, update *ext.Update) error {
		if message := update.EffectiveMessage; message != nil && message.Media != nil && mediautil.IsSupported(message.Media) {
			return dispatcher.ContinueGroups
		}
		return next(ctx, update)
	}
}

func handleSilentMode(next func(*ext.Context, *ext.Update) error, handler func(*ext.Context, *ext.Update) error) func(*ext.Context, *ext.Update) error {
	return func(ctx *ext.Context, update *ext.Update) error {
		userID := responsibleUserID(update)
		user, err := database.GetUserByChatID(ctx, userID)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetUserInfoFailed, map[string]any{
				"Error": err.Error(),
			})), nil)
			return dispatcher.EndGroups
		}
		if !user.Silent {
			return next(ctx, update)
		}
		if user.DefaultStorageID == 0 {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorDefaultStorageNotSet, nil)), nil)
			return next(ctx, update)
		}
		stor, err := storage.GetStorageByUserIDAndID(ctx, userID, user.DefaultStorageID)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{
				"Error": err.Error(),
			})), nil)
			return dispatcher.EndGroups
		}
		if user.DefaultDirID != 0 {
			dir, err := database.GetDirByID(ctx, user.DefaultDirID)
			if err != nil {
				ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetDirFailed, map[string]any{
					"Error": err.Error(),
				})), nil)
				return next(ctx, update)
			}
			ctx.Context = dirutil.WithContext(ctx.Context, dir)
		}
		ctx.Context = storage.WithContext(ctx.Context, stor)
		return handler(ctx, update)
	}
}
