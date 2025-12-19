package handlers

import (
	"fmt"
	"path"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
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
		logger.Errorf("Failed to build storage selection keyboard: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTelegraphErrorBuildStorageSelectKeyboardFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}

	eb := entity.Builder{}
	if err := styling.Perform(&eb,
		styling.Plain(i18n.T(i18nk.BotMsgTelegraphInfoTitlePrefix, nil)),
		styling.Code(result.Page.Title),
		styling.Plain(i18n.T(i18nk.BotMsgTelegraphInfoPicCountPrefix, nil)),
		styling.Code(fmt.Sprintf("%d", len(result.Pics))),
		styling.Plain(i18n.T(i18nk.BotMsgTelegraphInfoPromptSelectStorage, nil)),
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
	stor := storage.FromContext(ctx)
	msg, result, err := shortcut.GetTphPicsFromMessageWithReply(ctx, update)
	if err != nil {
		return err
	}
	userID := update.GetUserChat().GetID()
	dirpath := result.TphDir
	if p := dirutil.PathFromContext(ctx); p != "" {
		dirpath = path.Join(p, dirpath)
	}
	return shortcut.CreateAndAddtelegraphWithEdit(ctx, userID, result.Page, dirpath, result.Pics, stor, msg.ID)

}
