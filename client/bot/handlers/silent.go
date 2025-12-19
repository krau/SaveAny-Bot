package handlers

import (
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleSilentCmd(ctx *ext.Context, update *ext.Update) error {
	user, err := database.GetUserByChatID(ctx, update.GetUserChat().GetID())
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetUserInfoFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return nil
	}
	if !user.Silent && user.DefaultStorage == "" {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorDefaultStorageNotSet, nil)), nil)
		return nil
	}
	user.Silent = !user.Silent
	if err := database.UpdateUser(ctx, user); err != nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorUpdateUserInfoFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return nil
	}
	if user.Silent {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonInfoSilentModeOn, nil)), nil)
	} else {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonInfoSilentModeOff, nil)), nil)
	}
	return dispatcher.EndGroups
}

func handleSetDefaultCallback(ctx *ext.Context, update *ext.Update) error {
	dataid := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	data, ok := cache.Get[tcbdata.SetDefaultStorage](dataid)

	failedAnswer := func(message string) error {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   message,
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	if !ok {
		return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorDataExpired, nil))
	}
	userID := update.CallbackQuery.GetUserID()

	storageName := data.StorageName
	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, storageName)
	if err != nil {
		return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{
			"Error": err.Error(),
		}))
	}
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorGetUserInfoFailed, map[string]any{
			"Error": err.Error(),
		}))
	}
	var dir *database.Dir
	if data.DirID != 0 {
		// 已经选择了文件夹
		var err error
		dir, err = database.GetDirByID(ctx, data.DirID)
		if err != nil {
			return failedAnswer(i18n.T(i18nk.BotMsgDirErrorGetUserDirsFailed, nil))
		}
		user.DefaultDir = dir.ID
	} else {
		// 检查是否有可用的文件夹
		dirs, err := database.GetDirsByUserIDAndStorageName(ctx, user.ID, storageName)
		if err != nil {
			return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorGetDirFailed, map[string]any{
				"Error": err.Error(),
			}))
		}
		if len(dirs) > 0 {
			// 要求选择文件夹
			markup, err := msgelem.BuildSetDefaultDirMarkup(ctx, storageName, dirs)
			if err != nil {
				return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorBuildDirSelectKeyboardFailed, map[string]any{
					"Error": err.Error(),
				}))
			}
			ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
				ID:          update.CallbackQuery.GetMsgID(),
				Message:     i18n.T(i18nk.BotMsgCommonPromptSelectDefaultDir, nil),
				ReplyMarkup: markup,
			})
			return dispatcher.EndGroups
		}
	}
	user.DefaultStorage = selectedStorage.Name()
	if err := database.UpdateUser(ctx, user); err != nil {
		return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorUpdateUserInfoFailed, map[string]any{
			"Error": err.Error(),
		}))
	}
	msg := i18n.T(i18nk.BotMsgCommonInfoDefaultStorageSet, map[string]any{
		"Name": selectedStorage.Name(),
	})
	if dir != nil {
		msg = i18n.T(i18nk.BotMsgCommonInfoDefaultStorageWithDirSet, map[string]any{
			"Name": selectedStorage.Name(),
			"Dir":  strings.TrimPrefix(dir.Path, "/"),
		})
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: msg,
	})
	return dispatcher.EndGroups
}

func handleStorageCmd(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	storages := storage.GetUserStorages(ctx, userID)
	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorNoAvailableStorage, nil)), nil)
		return nil
	}
	markup, err := msgelem.BuildSetDefaultStorageMarkup(ctx, storages)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return nil
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonPromptSelectDefaultStorage, nil)), &ext.ReplyOpts{
		Markup: markup,
	})
	return dispatcher.EndGroups
}
