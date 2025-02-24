package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage"
)

func storageCmd(ctx *ext.Context, update *ext.Update) error {
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	storages := storage.GetUserStorages(user.ChatID)
	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}

	ctx.Reply(update, ext.ReplyTextString("请选择要设为默认的存储位置"), &ext.ReplyOpts{
		Markup: getSetDefaultStorageMarkup(user.ChatID, storages),
	})

	return dispatcher.EndGroups
}

func setDefaultStorage(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(string(update.CallbackQuery.Data), " ")
	userID, _ := strconv.Atoi(args[1])
	storageNameHash := args[2]
	if userID != int(update.CallbackQuery.GetUserID()) {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "你没有权限",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	storageName := storageHashName[storageNameHash]
	selectedStorage, err := storage.GetStorageByName(storageName)

	if err != nil {
		logger.L.Errorf("获取指定存储失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取指定存储失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	user, err := dao.GetUserByChatID(int64(userID))
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取用户失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	user.DefaultStorage = storageName
	if err := dao.UpdateUser(user); err != nil {
		logger.L.Errorf("Failed to update user: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "更新用户失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("已将 %s (%s) 设为默认存储位置", selectedStorage.Name(), selectedStorage.Type()),
		ID:      update.CallbackQuery.GetMsgID(),
	})
	return dispatcher.EndGroups
}
