package handlers

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleMessageLink(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	msgLinks := re.TgMessageLinkRegexp.FindAllString(update.EffectiveMessage.GetMessage(), -1)
	if len(msgLinks) == 0 {
		logger.Warn("no matched message links but called handleMessageLink")
		return dispatcher.ContinueGroups
	}
	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取消息..."), nil)
	if err != nil {
		logger.Errorf("failed to reply: %s", err)
		return dispatcher.EndGroups
	}

	files := make([]tfile.TGFile, 0, len(msgLinks))
	for _, link := range msgLinks {
		chatId, msgId, err := tgutil.ParseMessageLink(ctx, link)
		if err != nil {
			logger.Errorf("failed to parse message link %s: %s", link, err)
			continue
		}
		msg, err := tgutil.GetMessageByID(ctx, chatId, msgId)
		if err != nil {
			logger.Errorf("failed to get message by ID: %s", err)
			continue
		}
		media, ok := msg.GetMedia()
		if !ok {
			logger.Debugf("message %d has no media", msg.GetID())
			continue
		}
		file, err := tfile.FromMedia(media, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*msg)))
		if err != nil {
			logger.Errorf("failed to create file from media: %s", err)
			continue
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		ctx.Reply(update, ext.ReplyTextString("没有找到可保存的文件"), nil)
		return dispatcher.EndGroups
	}
	userId := update.GetUserChat().GetID()
	stors := storage.GetUserStorages(ctx, userId)
	if len(files) == 1 {
		req, err := msgelem.BuildAddOneSelectStorageMessage(ctx, stors, files[0], replied.ID)
		if err != nil {
			logger.Errorf("构建存储选择消息失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("构建存储选择消息失败: "+err.Error()), nil)
			return dispatcher.EndGroups
		}
		ctx.EditMessage(update.EffectiveChat().GetID(), req)
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:          replied.ID,
		Message:     "请选择存储位置",
		ReplyMarkup: msgelem.BuildAddBatchSelectStorageKeyboard(stors, files),
	})
	return dispatcher.EndGroups
}
