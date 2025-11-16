package handlers

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleMediaMessage(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	message := update.EffectiveMessage.Message
	groupID, isGroup := message.GetGroupedID()
	if isGroup && groupID != 0 {
		return handleGroupMediaMessage(ctx, update, message, groupID)
	}
	logger.Debugf("Got media: %s", message.Media.TypeName())
	userId := update.GetUserChat().GetID()
	userDB, err := database.GetUserByChatID(ctx, userId)
	if err != nil {
		return err
	}
	tfOpts := mediautil.TfileOptions(ctx, userDB, message)
	msg, file, err := shortcut.GetFileFromMessageWithReply(ctx, update, message, tfOpts...)
	if err != nil {
		return err
	}

	stors := storage.GetUserStorages(ctx, userId)
	req, err := msgelem.BuildAddOneSelectStorageMessage(ctx, stors, file, msg.ID)
	if err != nil {
		logger.Errorf("构建存储选择消息失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("构建存储选择消息失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), req)
	return dispatcher.EndGroups
}

func handleSilentSaveMedia(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(update, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	message := update.EffectiveMessage.Message
	groupID, isGroup := message.GetGroupedID()
	if isGroup && groupID != 0 {
		return handleGroupMediaMessage(ctx, update, message, groupID)
	}
	logger.Debugf("Got media: %s", message.Media.TypeName())
	userID := update.GetUserChat().GetID()
	userDB, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	tfOpts := mediautil.TfileOptions(ctx, userDB, message)
	msg, file, err := shortcut.GetFileFromMessageWithReply(ctx, update, message, tfOpts...)
	if err != nil {
		return err
	}
	return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userID, stor, dirutil.PathFromContext(ctx), file, msg.ID)
}
