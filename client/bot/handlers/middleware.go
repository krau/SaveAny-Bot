package handlers

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/storage"
)

// responsibleUserID returns the sender's ID. Callback queries carry it
// natively; message updates resolve it through the entity map.
func responsibleUserID(u *ext.Update) int64 {
	if u.CallbackQuery != nil {
		return u.CallbackQuery.GetUserID()
	}
	return u.GetUserChat().GetID()
}

func checkPermission(ctx *ext.Context, update *ext.Update) error {
	userID := responsibleUserID(update)
	if !slice.Contain(config.C().GetUsersID(), userID) {
		if cbq := update.CallbackQuery; cbq != nil {
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(cbq.GetQueryID(), i18n.T(i18nk.BotMsgCommonErrorNoPermission, nil)))
		} else {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorNoPermission, nil)), nil)
		}
		return dispatcher.EndGroups
	}

	return dispatcher.ContinueGroups
}

// withPermission wraps a callback handler with the same whitelist check used
// for message handlers (checkPermission).
func withPermission(handler func(*ext.Context, *ext.Update) error) func(*ext.Context, *ext.Update) error {
	return func(ctx *ext.Context, update *ext.Update) error {
		if err := checkPermission(ctx, update); err != nil {
			return err
		}
		return handler(ctx, update)
	}
}

func handleSilentMode(next func(*ext.Context, *ext.Update) error, handler func(*ext.Context, *ext.Update) error) func(*ext.Context, *ext.Update) error {
	return func(ctx *ext.Context, update *ext.Update) error {
		userID := update.GetUserChat().GetID()
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
		if user.DefaultStorage == "" {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorDefaultStorageNotSet, nil)), nil)
			return next(ctx, update)
		}
		stor, err := storage.GetStorageByUserIDAndName(ctx, userID, user.DefaultStorage)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{
				"Error": err.Error(),
			})), nil)
			return dispatcher.EndGroups
		}
		if user.DefaultDir != 0 {
			dir, err := database.GetDirByID(ctx, user.DefaultDir)
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
