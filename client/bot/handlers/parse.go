// 处理任意文本消息, 用于通用地从外部源下载文件

package handlers

import (
	"errors"
	"path"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/parsers"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleTextMessage(ctx *ext.Context, u *ext.Update) error {
	logger := log.FromContext(ctx)
	text := u.EffectiveMessage.Text
	entityUrls := tgutil.ExtractMessageEntityUrls(u.EffectiveMessage.Message)
	if len(entityUrls) > 0 {
		text += "\n" + strings.Join(entityUrls, "\n")
	}
	// read lines and remove empty lines & duplicates
	lines := strings.Split(text, "\n")
	seen := make(map[string]struct{})
	var processedLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		processedLines = append(processedLines, line)
	}
	source := strings.TrimSpace(strings.Join(processedLines, "\n"))
	ok, pser := parsers.CanHandle(source)
	if !ok {
		return dispatcher.EndGroups
	}
	msg, err := ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParseInfoParsing, nil)), nil)
	if err != nil {
		return err
	}

	item, err := pser.Parse(ctx, source)
	if errors.Is(err, parsers.ErrNoParserFound) {
		return dispatcher.EndGroups
	}
	if err != nil {
		logger.Error("Failed to parse text", "error", err)
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParseErrorParseTextFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	logger.Debug("Parsed item from text message", "title", item.Title, "url", item.URL)
	userID := u.GetUserChat().GetID()
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, userID), tcbdata.Add{
		TaskType:   tasktype.TaskTypeParseditem,
		ParsedItem: item,
	})
	if err != nil {
		logger.Errorf("Failed to build storage selection keyboard: %s", err)
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParseErrorBuildStorageSelectKeyboardFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	text, entities, err := msgelem.BuildParsedTextEntity(*item)
	if err != nil {
		logger.Errorf("Failed to build parsed text entity: %s", err)
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParseErrorBuildParsedTextEntityFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		Message:     text,
		ReplyMarkup: markup,
		Entities:    entities,
		ID:          msg.ID,
	})

	return dispatcher.EndGroups
}

func handleSilentSaveText(ctx *ext.Context, u *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
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
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParseErrorParseTextFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	logger.Debug("Parsed item from text message", "title", item.Title, "url", item.URL)
	userID := u.GetUserChat().GetID()
	text, entities, err := msgelem.BuildParsedTextEntity(*item)
	if err != nil {
		logger.Errorf("Failed to build parsed text entity: %s", err)
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParseErrorBuildParsedTextEntityFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	msg, err := ctx.SendMessage(userID, &tg.MessagesSendMessageRequest{
		Message:  text,
		Entities: entities,
		ReplyTo: &tg.InputReplyToMessage{
			ReplyToMsgID:  u.EffectiveMessage.ID,
			ReplyToPeerID: u.GetUserChat().AsInputPeer(),
		},
	})
	if err != nil {
		logger.Errorf("Failed to send message: %s", err)
		return dispatcher.EndGroups
	}
	dirPath := ""
	if len(item.Resources) > 1 {
		dirPath = fsutil.NormalizePathname(item.Title)
	}
	if p := dirutil.PathFromContext(ctx); p != "" {
		dirPath = path.Join(p, dirPath)
	}
	return shortcut.CreateAndAddParsedTaskWithEdit(ctx, stor, dirPath, item, msg.ID, userID)
}
