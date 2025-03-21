package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
)

func storageCmd(ctx *ext.Context, update *ext.Update) error {
	userChatID := update.GetUserChat().GetID()
	storages := storage.GetUserStorages(userChatID)
	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}
	markup, err := getSetDefaultStorageMarkup(userChatID, storages)
	if err != nil {
		common.Log.Errorf("Failed to get markup: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取存储位置失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("请选择要设为默认的存储位置"), &ext.ReplyOpts{
		Markup: markup,
	})
	return dispatcher.EndGroups
}

func setDefaultStorage(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(string(update.CallbackQuery.Data), " ")
	userID, _ := strconv.Atoi(args[1])
	if userID != int(update.CallbackQuery.GetUserID()) {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "你没有权限",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	cbDataId, _ := strconv.Atoi(args[2])
	storageName, err := dao.GetCallbackData(uint(cbDataId))
	if err != nil {
		common.Log.Errorf("获取回调数据失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取回调数据失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	selectedStorage, err := storage.GetStorageByName(storageName)

	if err != nil {
		common.Log.Errorf("获取指定存储失败: %s", err)
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
		common.Log.Errorf("Failed to get user: %s", err)
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
		common.Log.Errorf("Failed to update user: %s", err)
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
