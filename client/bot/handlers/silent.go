package handlers

import (
	"fmt"
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
		return failedAnswer("数据已过期")
	}
	userID := update.CallbackQuery.GetUserID()

	storageName := data.StorageName
	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, storageName)
	if err != nil {
		return failedAnswer("存储获取失败: " + err.Error())
	}
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return failedAnswer("获取用户信息失败: " + err.Error())
	}
	var dir *database.Dir
	if data.DirID != 0 {
		// 已经选择了文件夹
		var err error
		dir, err = database.GetDirByID(ctx, data.DirID)
		if err != nil {
			return failedAnswer("获取文件夹信息失败: " + err.Error())
		}
		user.DefaultDir = dir.ID
	} else {
		// 检查是否有可用的文件夹
		dirs, err := database.GetDirsByUserIDAndStorageName(ctx, user.ID, storageName)
		if err != nil {
			return failedAnswer("获取目录失败: " + err.Error())
		}
		if len(dirs) > 0 {
			// 要求选择文件夹
			markup, err := msgelem.BuildSetDefaultDirMarkup(ctx, storageName, dirs)
			if err != nil {
				return failedAnswer("构建目录选择失败: " + err.Error())
			}
			ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
				ID:          update.CallbackQuery.GetMsgID(),
				Message:     "请选择要保存到的默认文件夹",
				ReplyMarkup: markup,
			})
			return dispatcher.EndGroups
		}
	}
	user.DefaultStorage = selectedStorage.Name()
	if err := database.UpdateUser(ctx, user); err != nil {
		return failedAnswer("更新用户信息失败: " + err.Error())
	}
	msg := fmt.Sprintf("已将默认存储位置设为: %s", selectedStorage.Name())
	if dir != nil {
		msg += fmt.Sprintf(":/%s", strings.TrimPrefix(dir.Path, "/"))
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
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return nil
	}
	markup, err := msgelem.BuildSetDefaultStorageMarkup(ctx, storages)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("获取存储失败: "+err.Error()), nil)
		return nil
	}
	ctx.Reply(update, ext.ReplyTextString("请选择要设为默认的存储位置"), &ext.ReplyOpts{
		Markup: markup,
	})
	return dispatcher.EndGroups
}
