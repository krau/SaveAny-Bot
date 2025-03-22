package bot

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func handleFileMessage(ctx *ext.Context, update *ext.Update) error {
	common.Log.Trace("Got media: ", update.EffectiveMessage.Media.TypeName())
	supported, err := supportedMediaFilter(update.EffectiveMessage.Message)
	if err != nil {
		return err
	}
	if !supported {
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

	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		common.Log.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}
	media := update.EffectiveMessage.Media
	file, err := FileFromMedia(media, "")
	if err != nil {
		common.Log.Errorf("获取文件失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("获取文件失败: %s", err)), nil)
		return dispatcher.EndGroups
	}
	if file.FileName == "" {
		file.FileName = GenFileNameFromMessage(*update.EffectiveMessage.Message, file)
	}

	if err := dao.SaveReceivedFile(&dao.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      update.EffectiveMessage.ID,
		ReplyMessageID: msg.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	}); err != nil {
		common.Log.Errorf("添加接收的文件失败: %s", err)
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("添加接收的文件失败: %s", err),
			ID:      msg.ID,
		}); err != nil {
			common.Log.Errorf("编辑消息失败: %s", err)
		}
		return dispatcher.EndGroups
	}

	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, file.FileName, update.EffectiveChat().GetID(), update.EffectiveMessage.ID, msg.ID)
	}
	return HandleSilentAddTask(ctx, update, user, &types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		File:           file,
		StorageName:    user.DefaultStorage,
		FileChatID:     update.EffectiveChat().GetID(),
		ReplyMessageID: msg.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
		FileMessageID:  update.EffectiveMessage.ID,
		UserID:         user.ChatID,
	})
}
