// 处理任意文本消息, 用于通用地从外部源下载文件

package handlers

import (
	"errors"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/parsers"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleTextMessage(ctx *ext.Context, u *ext.Update) error {
	logger := log.FromContext(ctx)
	text := u.EffectiveMessage.Text
	item, err := parsers.ParseWithContext(ctx, text)
	if errors.Is(err, parsers.ErrNoParserFound) {
		return dispatcher.EndGroups
	}
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
	text, entities, err := msgelem.BuildParsedTextEntity(*item)
	if err != nil {
		logger.Errorf("Failed to build parsed text entity: %s", err)
		ctx.Reply(u, ext.ReplyTextString("Failed to build parsed text entity: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	ctx.SendMessage(userID, &tg.MessagesSendMessageRequest{
		Message:     text,
		ReplyMarkup: markup,
		Entities:    entities,
	})

	return dispatcher.EndGroups
}

func handleSilentSaveText(ctx *ext.Context, u *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(u, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	text := u.EffectiveMessage.Text
	if text == "" {
		return dispatcher.EndGroups
	}
	item, err := parsers.ParseWithContext(ctx, text)
	if errors.Is(err, parsers.ErrNoParserFound) {
		return dispatcher.EndGroups
	}
	if err != nil {
		logger.Error("Failed to parse text", "error", err)
		ctx.Reply(u, ext.ReplyTextString("Failed to parse text: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	logger.Debug("Parsed item from text message", "text", text, "item", item)
	userID := u.GetUserChat().GetID()
	text, entities, err := msgelem.BuildParsedTextEntity(*item)
	if err != nil {
		logger.Errorf("Failed to build parsed text entity: %s", err)
		ctx.Reply(u, ext.ReplyTextString("Failed to build parsed text entity: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	msg, err := ctx.SendMessage(userID, &tg.MessagesSendMessageRequest{
		Message:  text,
		Entities: entities,
	})
	if err != nil {
		logger.Errorf("Failed to send message: %s", err)
		return dispatcher.EndGroups
	}
	return shortcut.CreateAndAddParsedTaskWithEdit(ctx, stor, "", item, msg.ID, userID)
}
