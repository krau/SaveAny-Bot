package handlers

import (
	"bytes"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/parsers"
)

func handleParserCmd(ctx *ext.Context, u *ext.Update) error {
	args := strings.Split(u.EffectiveMessage.Text, " ")
	help := `
用法:

/parser install <回复一个文件> - 安装解析器
`
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
		ctx.Reply(u, ext.ReplyTextString("解析器插件功能未启用"), nil)
		return dispatcher.EndGroups
	}
	if u.EffectiveMessage.ReplyToMessage == nil || u.EffectiveMessage.ReplyToMessage.Media == nil {
		ctx.Reply(u, ext.ReplyTextString("请回复一个包含解析器文件的消息"), nil)
		return dispatcher.EndGroups
	}
	media := u.EffectiveMessage.ReplyToMessage.Media
	document, ok := media.(*tg.MessageMediaDocument)
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("回复的消息不包含有效的文件"), nil)
		return dispatcher.EndGroups
	}
	value, ok := document.GetDocument()
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("回复的消息不包含有效的文件"), nil)
		return dispatcher.EndGroups
	}
	doc, ok := value.AsNotEmpty()
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("回复的消息不包含有效的文件"), nil)
		return dispatcher.EndGroups
	}
	if !strings.HasPrefix(doc.MimeType, "text/") {
		ctx.Reply(u, ext.ReplyTextString("错误的文件类型"), nil)
		return dispatcher.EndGroups
	}
	if doc.Size > 1024*1024*10 {
		ctx.Reply(u, ext.ReplyTextString("文件过大"), nil)
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
		ctx.Reply(u, ext.ReplyTextString("无法获取文件名"), nil)
		return dispatcher.EndGroups
	}
	if !strings.HasSuffix(fileName, ".js") {
		ctx.Reply(u, ext.ReplyTextString("仅支持 .js 文件作为解析器"), nil)
		return dispatcher.EndGroups
	}
	data := bytes.NewBuffer(nil)
	_, err := ctx.DownloadMedia(media, ext.DownloadOutputStream{Writer: data}, nil)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString("文件下载失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	if err := parsers.AddPlugin(ctx, data.String(), fileName); err != nil {
		ctx.Reply(u, ext.ReplyTextString("插件安装失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(u, ext.ReplyTextString("插件安装成功: "+fileName), nil)
	return dispatcher.EndGroups
}
