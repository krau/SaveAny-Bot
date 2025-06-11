package bot

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/database"
)

func handleFileMessage(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Got media: ", update.EffectiveMessage.Media.TypeName())
	supported, err := supportedMediaFilter(update.EffectiveMessage.Message)
	if err != nil {
		return err
	}
	if !supported {
		return dispatcher.EndGroups
	}

	_, err = database.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		logger.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}

	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}
	media := update.EffectiveMessage.Media
	file, err := FileFromMedia(media, "")
	if err != nil {
		logger.Errorf("获取文件失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("获取文件失败: %s", err)), nil)
		return dispatcher.EndGroups
	}
	if file.FileName == "" {
		file.FileName = GenFileNameFromMessage(*update.EffectiveMessage.Message, file)
	}

	_, err = database.SaveReceivedFile(&database.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      update.EffectiveMessage.ID,
		ReplyMessageID: msg.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	})
	if err != nil {
		logger.Errorf("添加接收的文件失败: %s", err)
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("添加接收的文件失败: %s", err),
			ID:      msg.ID,
		}); err != nil {
			logger.Errorf("编辑消息失败: %s", err)
		}
		return dispatcher.EndGroups
	}
	return ProvideSelectMessage(ctx, update, file.FileName, update.EffectiveChat().GetID(), update.EffectiveMessage.ID, msg.ID)
}
