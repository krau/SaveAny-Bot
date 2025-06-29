package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleMessageLink(ctx *ext.Context, update *ext.Update) error {
	replied, files, editReplied, err := shortcut.GetFilesFromUpdateLinkMessageWithReplyEdit(ctx, update)
	if err != nil {
		return err
	}
	logger := log.FromContext(ctx)
	userId := update.GetUserChat().GetID()
	stors := storage.GetUserStorages(ctx, userId)
	if len(files) == 1 {
		req, err := msgelem.BuildAddOneSelectStorageMessage(ctx, stors, files[0], replied.ID)
		if err != nil {
			logger.Errorf("构建存储选择消息失败: %s", err)
			editReplied("构建存储选择消息失败: "+err.Error(), nil)
			return dispatcher.EndGroups
		}
		ctx.EditMessage(update.EffectiveChat().GetID(), req)
		return dispatcher.EndGroups
	}
	markup, err := msgelem.BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
		Files: files,
	})
	if err != nil {
		logger.Errorf("构建存储选择键盘失败: %s", err)
		editReplied("构建存储选择键盘失败: "+err.Error(), nil)
		return dispatcher.EndGroups
	}
	editReplied(fmt.Sprintf("找到 %d 个文件, 请选择存储位置", len(files)), markup)
	return dispatcher.EndGroups
}

func handleSilentSaveLink(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(update, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	replied, files, _, err := shortcut.GetFilesFromUpdateLinkMessageWithReplyEdit(ctx, update)
	if err != nil {
		return err
	}
	userId := update.GetUserChat().GetID()
	if len(files) == 1 {
		return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userId, stor, "", files[0], replied.ID)
	}
	return shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, userId, stor, "", files, replied.ID)
}
