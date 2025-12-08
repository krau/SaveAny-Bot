package handlers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleDlCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString("用法: /dl <链接1> <链接2> ..."), nil)
		return nil
	}
	links := args[1:]
	for i, link := range links {
		links[i] = strings.TrimSpace(link)
		u, err := url.Parse(link)
		if err != nil || u.Scheme == "" || u.Host == "" {
			logger.Warn("invaild link", link)
			links[i] = ""
		}
	}
	links = slice.Compact(links)
	if len(links) == 0 {
		ctx.Reply(update, ext.ReplyTextString("没有有效的链接可供下载"), nil)
		return nil
	}
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, update.GetUserChat().GetID()), tcbdata.Add{
		TaskType:    tasktype.TaskTypeDirectlinks,
		DirectLinks: links,
	})
	if err != nil {
		return err
	}
	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("共 %d 个文件, 请选择存储位置", len(links))), &ext.ReplyOpts{
		Markup: markup,
	})
	return nil
}
