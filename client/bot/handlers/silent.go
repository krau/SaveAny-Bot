package handlers

import (
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleSilentCmd(ctx *ext.Context, update *ext.Update) error {
	user, err := database.GetUserByChatID(ctx, update.GetUserChat().GetID())
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("获取用户信息失败: "+err.Error()), nil)
		return nil
	}
	if !user.Silent && user.DefaultStorage == "" {
		ctx.Reply(update, ext.ReplyTextString("请先使用 /storage 设置默认存储位置"), nil)
		return nil
	}
	user.Silent = !user.Silent
	if err := database.UpdateUser(ctx, user); err != nil {
		ctx.Reply(update, ext.ReplyTextString("更新用户信息失败: "+err.Error()), nil)
		return nil
	}
	responseText := "已" + map[bool]string{true: "开启", false: "关闭"}[user.Silent] + "静默模式"
	ctx.Reply(update, ext.ReplyTextString(responseText), nil)
	return dispatcher.EndGroups
}

func handleSetDefaultCallback(ctx *ext.Context, update *ext.Update) error {
	dataid := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	data, ok := cache.Get[tcbdata.SetDefaultStorage](dataid)
	if !ok {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "数据已过期",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	userID := update.CallbackQuery.GetUserID()

	storageName := data.StorageName
	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, storageName)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "存储获取失败: " + err.Error(),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "获取用户信息失败: " + err.Error(),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	user.DefaultStorage = selectedStorage.Name()
	if err := database.UpdateUser(ctx, user); err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "更新用户信息失败: " + err.Error(),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: "已将默认存储位置设置为: " + selectedStorage.Name(),
	})
	return dispatcher.EndGroups
}

func handleStorageCmd(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	storages := storage.GetUserStorages(ctx, userID)
	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return nil
	}
	markup, err := msgelem.BuildSetDefaultStorageMarkup(ctx, userID, storages)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("获取存储失败: "+err.Error()), nil)
		return nil
	}
	ctx.Reply(update, ext.ReplyTextString("请选择要设为默认的存储位置"), &ext.ReplyOpts{
		Markup: markup,
	})
	return dispatcher.EndGroups
}
