package handlers

import (
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func handleMessageLink(ctx *ext.Context, update *ext.Update) error {
	replied, files, editReplied, err := shortcut.GetFilesFromUpdateLinkMessageWithReplyEdit(ctx, update)
	if err != nil {
		return err
	}
	logger := log.FromContext(ctx)
	userId := responsibleUserID(update)
	stors := storage.GetUserStorages(ctx, userId)
	if len(files) == 1 {
		req, err := msgelem.BuildAddOneSelectStorageMessage(ctx, stors, files[0], replied.ID)
		if err != nil {
			logger.Errorf("Failed to build storage selection message: %s", err)
			editReplied(i18n.T(i18nk.BotMsgCommonErrorBuildStorageSelectMessageFailed, map[string]any{
				"Error": err.Error(),
			}), nil)
			return dispatcher.EndGroups
		}
		ctx.EditMessage(update.EffectiveChat().GetID(), req)
		return dispatcher.EndGroups
	}
	markup, err := msgelem.BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
		Files: files,
	})
	if err != nil {
		logger.Errorf("Failed to build storage selection keyboard: %s", err)
		editReplied(i18n.T(i18nk.BotMsgCommonErrorBuildStorageSelectKeyboardFailed, map[string]any{
			"Error": err.Error(),
		}), nil)
		return dispatcher.EndGroups
	}
	editReplied(i18n.T(i18nk.BotMsgCommonInfoFoundFilesSelectStorage, map[string]any{
		"Count": len(files),
	}), markup)
	return dispatcher.EndGroups
}

func handleSilentSaveLink(ctx *ext.Context, update *ext.Update) error {
	stor := storage.FromContext(ctx)
	replied, files, _, err := shortcut.GetFilesFromUpdateLinkMessageWithReplyEdit(ctx, update)
	if err != nil {
		return err
	}
	userId := responsibleUserID(update)
	if len(files) == 1 {
		return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userId, chatID(update), stor, dirutil.PathFromContext(ctx), files[0], replied.ID)
	}
	return shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, userId, chatID(update), stor, dirutil.PathFromContext(ctx), files, replied.ID)
}
