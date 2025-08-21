// 处理任意文本消息, 用于通用地从外部源下载文件

package handlers

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/parsers"
)

func handleTextMessage(ctx *ext.Context, u *ext.Update) error {
	logger := log.FromContext(ctx)
	text := u.EffectiveMessage.Text
	item, err := parsers.ParseWithContext(ctx, text)
	if err == nil {
		logger.Debug("Parsed item from text", "item", item)
		ctx.Reply(u, ext.ReplyTextString("Parsed item: "+item.Title), nil)
	} else {
		logger.Error("Failed to parse text", "error", err)
		ctx.Reply(u, ext.ReplyTextString("Failed to parse text: "+err.Error()), nil)
	}
	return dispatcher.EndGroups
}

func handleSilentSaveText(ctx *ext.Context, u *ext.Update) error {
	// [TODO]
	return nil
}
