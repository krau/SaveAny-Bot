package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func handleDirCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strutil.ParseArgsRespectQuotes(update.EffectiveMessage.Text)
	userChatID := responsibleUserID(update)
	dirs, err := database.GetUserDirsByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user directories: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorGetUserDirsFailed)), nil)
		return dispatcher.EndGroups
	}
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(dirMenuText(dirs)), &ext.ReplyOpts{Markup: dirMenuMarkup(dirs)})
		return dispatcher.EndGroups
	}
	user, err := database.GetUserByChatID(ctx, responsibleUserID(update))
	if err != nil {
		logger.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorGetUserFailed)), nil)
		return dispatcher.EndGroups
	}
	switch args[1] {
	case "add", "new":
		// /dirs add local1 path/to/dir
		if len(args) < 4 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
			return dispatcher.EndGroups
		}
		selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, user.ChatID, args[2])
		if err != nil {
			logger.Errorf("Failed to get storage %q: %s", args[2], err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed,
				tgutil.EscapeHTMLTemplateData(map[string]any{"Error": err.Error()}))), nil)
			return dispatcher.EndGroups
		}

		if err := database.CreateDirForUser(ctx, user.ID, storage.ID(selectedStorage), args[3]); err != nil {
			logger.Errorf("Failed to create directory: %s", err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorCreateDirFailed)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirInfoCreateDirSuccess)), nil)
	case "del", "remove":
		// /dirs del 3
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
			return dispatcher.EndGroups
		}
		dirID, err := strconv.Atoi(args[2])
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorInvalidDirId)), nil)
			return dispatcher.EndGroups
		}
		if err := database.DeleteDirByID(ctx, uint(dirID), user.ID); err != nil {
			logger.Errorf("Failed to delete directory: %s", err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorDeleteDirFailed)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirInfoDeleteDirSuccess)), nil)
	default:
		ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
	}
	return dispatcher.EndGroups
}

func dirMenuText(dirs []database.Dir) string {
	if len(dirs) == 0 {
		return i18n.T(i18nk.BotMsgDirMenuEmpty) + "\n\n" + i18n.T(i18nk.BotMsgDirMenuHint)
	}
	var b strings.Builder
	b.WriteString(i18n.T(i18nk.BotMsgDirMenuTitle))
	b.WriteString("\n\n")
	b.WriteString(i18n.T(i18nk.BotMsgDirMenuHint))
	for _, dir := range dirs {
		storageName := config.C().GetStorageNameByID(dir.StorageID)
		if storageName == "" {
			storageName = fmt.Sprintf("#%d", dir.StorageID)
		}
		fmt.Fprintf(&b, "\n\n#%d %s - %s", dir.ID, storageName, dir.Path)
	}
	return b.String()
}

func dirMenuMarkup(dirs []database.Dir) *tg.ReplyInlineMarkup {
	if len(dirs) == 0 {
		return nil
	}
	rows := make([]tg.KeyboardInlineButtonRow, 0, len(dirs))
	for _, dir := range dirs {
		rows = append(rows, tg.KeyboardInlineButtonRow{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgDirButtonDelete), fmt.Appendf(nil, "%s del %d", tcbdata.TypeDir, dir.ID)),
		}})
	}
	return &tg.ReplyInlineMarkup{Rows: rows}
}

func handleDirCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	if args[1] == "del" && len(args) >= 3 {
		dirID, err := strconv.Atoi(args[2])
		if err != nil {
			return err
		}
		if err := database.DeleteDirByID(ctx, uint(dirID), user.ID); err != nil {
			return err
		}
	}
	dirs, err := database.GetUserDirsByChatID(ctx, userID)
	if err != nil {
		return err
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     dirMenuText(dirs),
		ReplyMarkup: dirMenuMarkup(dirs),
	})
	return dispatcher.EndGroups
}
