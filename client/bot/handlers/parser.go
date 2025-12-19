package handlers

import (
	"bytes"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/parsers"
)

func handleParserCmd(ctx *ext.Context, u *ext.Update) error {
	args := strings.Split(u.EffectiveMessage.Text, " ")
	help := i18n.T(i18nk.BotMsgParserHelpText, nil)
	if len(args) < 2 {
		ctx.Reply(u, ext.ReplyTextString(help), nil)
		return nil
	}
	switch args[1] {
	// case "list":
	// 	return handleParserListCmd(ctx, u)
	case "install":
		return handleParserInstallCmd(ctx, u)
	// case "uninstall":
	// return handleParserUninstallCmd(ctx, u)
	default:
	}
	return dispatcher.EndGroups
}

func handleParserInstallCmd(ctx *ext.Context, u *ext.Update) error {
	if !config.C().Parser.PluginEnable {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserPluginNotEnabled, nil)), nil)
		return dispatcher.EndGroups
	}
	if u.EffectiveMessage.ReplyToMessage == nil || u.EffectiveMessage.ReplyToMessage.Media == nil {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserPromptReplyWithParserFile, nil)), nil)
		return dispatcher.EndGroups
	}
	media := u.EffectiveMessage.ReplyToMessage.Media
	document, ok := media.(*tg.MessageMediaDocument)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorNoValidFileInReply, nil)), nil)
		return dispatcher.EndGroups
	}
	value, ok := document.GetDocument()
	if !ok {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorNoValidFileInReply, nil)), nil)
		return dispatcher.EndGroups
	}
	doc, ok := value.AsNotEmpty()
	if !ok {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorNoValidFileInReply, nil)), nil)
		return dispatcher.EndGroups
	}
	if !strings.HasPrefix(doc.MimeType, "text/") {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorWrongFileType, nil)), nil)
		return dispatcher.EndGroups
	}
	if doc.Size > 1024*1024*10 {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorFileTooLarge, nil)), nil)
		return dispatcher.EndGroups
	}
	var fileName string
	for _, attr := range doc.Attributes {
		if fileNameAttr, ok := attr.(*tg.DocumentAttributeFilename); ok {
			fileName = fileNameAttr.FileName
			break
		}
	}
	if fileName == "" {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorGetFilenameFailed, nil)), nil)
		return dispatcher.EndGroups
	}
	if !strings.HasSuffix(fileName, ".js") {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorOnlyJsSupported, nil)), nil)
		return dispatcher.EndGroups
	}
	data := bytes.NewBuffer(nil)
	_, err := ctx.DownloadMedia(media, ext.DownloadOutputStream{Writer: data}, nil)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorDownloadFileFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	if err := parsers.AddPlugin(ctx, data.String(), fileName); err != nil {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserErrorInstallPluginFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgParserInfoInstallPluginSuccess, map[string]any{
		"Name": fileName,
	})), nil)
	return dispatcher.EndGroups
}
