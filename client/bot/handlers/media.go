package handlers

import (
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func handleMediaMessage(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	message := update.EffectiveMessage.Message
	groupID, isGroup := message.GetGroupedID()
	if isGroup && groupID != 0 {
		return handleGroupMediaMessage(ctx, update, message, groupID)
	}
	logger.Debugf("Got media: %s", message.Media.TypeName())
	userId := responsibleUserID(update)
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
		logger.Errorf("Failed to build storage selection message: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorBuildStorageSelectMessageFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), req)
	return dispatcher.EndGroups
}

func handleSilentSaveMedia(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	message := update.EffectiveMessage.Message
	groupID, isGroup := message.GetGroupedID()
	if isGroup && groupID != 0 {
		return handleGroupMediaMessage(ctx, update, message, groupID)
	}
	logger.Debugf("Got media: %s", message.Media.TypeName())
	userID := responsibleUserID(update)
	userDB, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	tfOpts := mediautil.TfileOptions(ctx, userDB, message)
	msg, file, err := shortcut.GetFileFromMessageWithReply(ctx, update, message, tfOpts...)
	if err != nil {
		return err
	}
	return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userID, chatID(update), stor, dirutil.PathFromContext(ctx), file, msg.ID)
}
