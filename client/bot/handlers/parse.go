// 处理任意文本消息, 用于通用地从外部源下载文件

package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/parsers"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleTextMessage(ctx *ext.Context, u *ext.Update) error {
	logger := log.FromContext(ctx)
	text := u.EffectiveMessage.Text
	msg, err := ctx.Reply(u, ext.ReplyTextString("正在尝试解析..."), nil)
	if err != nil {
		logger.Errorf("Failed to reply to message: %s", err)
		return dispatcher.EndGroups
	}
	item, err := parsers.ParseWithContext(ctx, text)
	if err != nil {
		logger.Error("Failed to parse text", "error", err)
		ctx.Reply(u, ext.ReplyTextString("Failed to parse text: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	logger.Debug("Parsed item from text message", "text", text, "item", item)
	userID := u.GetUserChat().GetID()
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, userID), tcbdata.Add{
		TaskType:   tasktype.TaskTypeParseditem,
		ParsedItem: item,
	})
	if err != nil {
		logger.Errorf("Failed to build storage selection keyboard: %s", err)
		ctx.Reply(u, ext.ReplyTextString("Failed to build storage selection keyboard: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	eb := entity.Builder{}
	if err := styling.Perform(&eb,
		styling.Plain("标题: "),
		styling.Code(item.Title),
		styling.Plain("\n文件数量: "),
		styling.Code(fmt.Sprintf("%d", len(item.Resources))),
		styling.Plain("\n预计总大小: "),
		styling.Code(fmt.Sprintf("%.2f MB", func() float64 {
			var totalSize int64
			for _, res := range item.Resources {
				totalSize += res.Size
			}
			return float64(totalSize) / 1024 / 1024
		}())),
		styling.Plain("\n请选择存储位置"),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
		return dispatcher.EndGroups
	}
	text, entities := eb.Complete()
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		Message:     text,
		ID:          msg.ID,
		ReplyMarkup: markup,
		Entities:    entities,
	})

	return dispatcher.EndGroups
}

func handleSilentSaveText(ctx *ext.Context, u *ext.Update) error {
	// [TODO]
	return nil
}
