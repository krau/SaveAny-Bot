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
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleTelegraphUrlMessage(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)

	msg, result, err := shortcut.GetTphPicsFromMessageWithReply(ctx, update)
	if err != nil {
		return err
	}
	userID := update.GetUserChat().GetID()
	stors := storage.GetUserStorages(ctx, userID)
	markup, err := msgelem.BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
		TaskType:    tasktype.TaskTypeTphpics,
		TphPageNode: result.Page,
		TphDirPath:  result.TphDir,
		TphPics:     result.Pics,
	})
	if err != nil {
		logger.Errorf("构建存储选择键盘失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("构建存储选择键盘失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}

	eb := entity.Builder{}
	if err := styling.Perform(&eb,
		styling.Plain("标题: "),
		styling.Code(result.Page.Title),
		styling.Plain("\n图片数量: "),
		styling.Code(fmt.Sprintf("%d", len(result.Pics))),
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
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(update, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	msg, result, err := shortcut.GetTphPicsFromMessageWithReply(ctx, update)
	if err != nil {
		return err
	}
	userID := update.GetUserChat().GetID()
	return shortcut.CreateAndAddtelegraphWithEdit(ctx, userID, result.Page, result.TphDir, result.Pics, stor, msg.ID)

}
