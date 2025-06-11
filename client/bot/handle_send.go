package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	tgtypes "github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
)

func copyMediaToChat(ctx *ext.Context, msg *tg.Message, chatID int64) (*tgtypes.Message, error) {
	media, ok := msg.GetMedia()
	if !ok {
		return nil, fmt.Errorf("获取媒体失败")
	}

	req := &tg.MessagesSendMediaRequest{
		InvertMedia: msg.InvertMedia,
		Message:     msg.Message,
	}

	switch m := media.(type) {
	case *tg.MessageMediaDocument:
		document, ok := m.Document.AsNotEmpty()
		if !ok {
			return nil, ErrEmptyDocument
		}
		inputMedia := &tg.InputMediaDocument{
			ID: document.AsInput(),
		}
		inputMedia.SetFlags()
		req.Media = inputMedia

	case *tg.MessageMediaPhoto:
		photo, ok := m.Photo.AsNotEmpty()
		if !ok {
			return nil, ErrEmptyPhoto
		}
		inputMedia := &tg.InputMediaPhoto{
			ID: photo.AsInput(),
		}
		inputMedia.SetFlags()
		req.Media = inputMedia

	default:
		return nil, fmt.Errorf("不支持的媒体类型: %T", media)
	}

	req.SetEntities(msg.Entities)
	req.SetFlags()

	return ctx.SendMedia(chatID, req)
}

func sendFileToTelegram(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(string(update.CallbackQuery.Data), " ")
	if len(args) < 3 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "参数错误",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	fileChatID, _ := strconv.Atoi(args[1])
	fileMessageID, _ := strconv.Atoi(args[2])
	fileMessage, err := GetTGMessage(ctx, int64(fileChatID), fileMessageID)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "无法获取文件消息",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	_, err = copyMediaToChat(ctx, fileMessage, update.EffectiveChat().GetID())
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   fmt.Sprintf("发送文件失败: %s", err),
			CacheTime: 5,
		})
	} else {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.QueryID,
		})
	}
	return dispatcher.EndGroups
}
