package bot

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func saveCmd(ctx *ext.Context, update *ext.Update) error {
	res, ok := update.EffectiveMessage.GetReplyTo()
	if !ok || res == nil {
		ctx.Reply(update, ext.ReplyTextString("请回复要保存的文件"), nil)
		return dispatcher.EndGroups
	}
	replyHeader, ok := res.(*tg.MessageReplyHeader)
	if !ok {
		ctx.Reply(update, ext.ReplyTextString("请回复要保存的文件"), nil)
		return dispatcher.EndGroups
	}
	replyToMsgID, ok := replyHeader.GetReplyToMsgID()
	if !ok {
		ctx.Reply(update, ext.ReplyTextString("请回复要保存的文件"), nil)
		return dispatcher.EndGroups
	}

	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}

	storages := storage.GetUserStorages(user.ChatID)

	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}

	msg, err := GetTGMessage(ctx, update.EffectiveChat().GetID(), replyToMsgID)
	if err != nil {
		common.Log.Errorf("获取消息失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("无法获取消息"), nil)
		return dispatcher.EndGroups
	}

	supported, _ := supportedMediaFilter(msg)
	if !supported {
		ctx.Reply(update, ext.ReplyTextString("不支持的消息类型或消息中没有文件"), nil)
		return dispatcher.EndGroups
	}

	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		common.Log.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}

	cmdText := update.EffectiveMessage.Text
	customFileName := strings.TrimSpace(strings.TrimPrefix(cmdText, "/save"))

	file, err := FileFromMessage(ctx, update.EffectiveChat().GetID(), msg.ID, customFileName)
	if err != nil {
		common.Log.Errorf("获取文件失败: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("获取文件失败: %s", err),
			ID:      replied.ID,
		})
		return dispatcher.EndGroups
	}

	if file.FileName == "" {
		file.FileName = GenFileNameFromMessage(*msg, file)
	}
	receivedFile := &dao.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      replyToMsgID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	}

	if err := dao.SaveReceivedFile(receivedFile); err != nil {
		common.Log.Errorf("保存接收的文件失败: %s", err)
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("保存接收的文件失败: %s", err),
			ID:      replied.ID,
		}); err != nil {
			common.Log.Errorf("编辑消息失败: %s", err)
		}
		return dispatcher.EndGroups
	}
	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, file.FileName, update.EffectiveChat().GetID(), msg.ID, replied.ID)
	}
	return HandleSilentAddTask(ctx, update, user, &types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		File:           file,
		StorageName:    user.DefaultStorage,
		FileChatID:     update.EffectiveChat().GetID(),
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
		FileMessageID:  msg.ID,
		UserID:         user.ChatID,
	})
}
