package handlers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/common/utils/tphutil"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleTelegraphUrlMessage(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	tphurl := re.TelegraphUrlRegexp.FindString(update.EffectiveMessage.GetMessage()) // TODO: batch urls
	if tphurl == "" {
		logger.Warnf("No telegraph url found but called handleTelegraph")
		return dispatcher.ContinueGroups
	}
	pagepath := strings.Split(tphurl, "/")[len(strings.Split(tphurl, "/"))-1]
	tphdir, err := url.PathUnescape(pagepath)
	if err != nil {
		logger.Errorf("Failed to unescape telegraph path: %s", err)
		ctx.Reply(update, ext.ReplyTextString("解析 telegraph 路径失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取 telegraph 页面..."), nil)
	if err != nil {
		logger.Errorf("Failed to reply to update: %s", err)
		return dispatcher.EndGroups
	}
	page, err := tphutil.DefaultClient().GetPage(ctx, pagepath)
	if err != nil {
		logger.Errorf("Failed to get telegraph page: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取 telegraph 页面失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	imgs := make([]string, 0)
	for _, elem := range page.Content {
		var node telegraph.NodeElement
		data, err := json.Marshal(elem)
		if err != nil {
			logger.Errorf("Failed to marshal element: %s", err)
			continue
		}
		err = json.Unmarshal(data, &node)
		if err != nil {
			logger.Errorf("Failed to unmarshal element: %s", err)
			continue
		}

		if len(node.Children) != 0 {
			for _, child := range node.Children {
				imgs = append(imgs, tphutil.GetNodeImages(child)...)
			}
		}
		if node.Tag == "img" {
			if src, ok := node.Attrs["src"]; ok {
				imgs = append(imgs, src)
			}
		}
	}
	if len(imgs) == 0 {
		logger.Warn("No images found in telegraph page")
		ctx.Reply(update, ext.ReplyTextString("在 telegraph 页面中未找到图片"), nil)
		return dispatcher.EndGroups
	}
	userID := update.GetUserChat().GetID()
	stors := storage.GetUserStorages(ctx, userID)
	markup, err := msgelem.BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
		TaskType:    tasktype.TaskTypeTphpics,
		TphPageNode: page,
		TphDirPath:  tphdir,
		TphPics:     imgs,
	})
	if err != nil {
		logger.Errorf("构建存储选择键盘失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("构建存储选择键盘失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}

	eb := entity.Builder{}
	if err := styling.Perform(&eb,
		styling.Plain("标题: "),
		styling.Code(page.Title),
		styling.Plain("\n图片数量: "),
		styling.Code(fmt.Sprintf("%d", len(imgs))),
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

func handleSilentSaveTelegraph(ctx *ext.Context, update *ext.Update) error {
	panic("handleSilentSaveTelegraph is not implemented")
}
