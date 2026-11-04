package handlers

import (
	"fmt"
	"strings"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func handleSetDefaultCallback(ctx *ext.Context, update *ext.Update) error {
	dataParts := strings.Split(string(update.CallbackQuery.Data), " ")
	if len(dataParts) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	dataid := dataParts[1]
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

	storageID := data.StorageID
	selectedStorage, err := storage.GetStorageByUserIDAndID(ctx, userID, storageID)
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
		var err error
		dir, err = database.GetDirByID(ctx, data.DirID)
		if err != nil {
			return failedAnswer(i18n.T(i18nk.BotMsgDirErrorGetUserDirsFailed, nil))
		}
		user.DefaultDirID = dir.ID
	} else {
		dirs, err := database.GetDirsByUserIDAndStorageID(ctx, user.ID, storageID)
		if err != nil {
			return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorGetDirFailed, map[string]any{
				"Error": err.Error(),
			}))
		}
		if len(dirs) > 0 {
			markup, err := msgelem.BuildSetDefaultDirMarkup(ctx, storageID, dirs)
			if err != nil {
				return failedAnswer(i18n.T(i18nk.BotMsgCommonErrorBuildDirSelectKeyboardFailed, map[string]any{
					"Error": err.Error(),
				}))
			}
			ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
				ID:          update.CallbackQuery.GetMsgID(),
				Message:     i18n.T(i18nk.BotMsgCommonPromptSelectDefaultDir, nil),
				ReplyMarkup: markup,
			})
			return dispatcher.EndGroups
		}
	}
	user.DefaultStorageID = storageID
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
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: msg,
	})
	return dispatcher.EndGroups
}
