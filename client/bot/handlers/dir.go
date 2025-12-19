package handlers

import (
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleDirCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	userChatID := update.GetUserChat().GetID()
	dirs, err := database.GetUserDirsByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user directories: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorGetUserDirsFailed)), nil)
		return dispatcher.EndGroups
	}
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
		return dispatcher.EndGroups
	}
	user, err := database.GetUserByChatID(ctx, update.GetUserChat().GetID())
	if err != nil {
		logger.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorGetUserFailed)), nil)
		return dispatcher.EndGroups
	}
	switch args[1] {
	case "add":
		// /dir add local1 path/to/dir
		if len(args) < 4 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
			return dispatcher.EndGroups
		}
		if _, err := storage.GetStorageByUserIDAndName(ctx, user.ChatID, args[2]); err != nil {
			ctx.Reply(update, ext.ReplyTextString(err.Error()), nil)
			return dispatcher.EndGroups
		}

		if err := database.CreateDirForUser(ctx, user.ID, args[2], args[3]); err != nil {
			logger.Errorf("Failed to create directory: %s", err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorCreateDirFailed)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirInfoCreateDirSuccess)), nil)
	case "del":
		// /dir del 3
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
			return dispatcher.EndGroups
		}
		dirID, err := strconv.Atoi(args[2])
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorInvalidDirId)), nil)
			return dispatcher.EndGroups
		}
		if err := database.DeleteDirByID(ctx, uint(dirID)); err != nil {
			logger.Errorf("Failed to delete directory: %s", err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorDeleteDirFailed)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirInfoDeleteDirSuccess)), nil)
	default:
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgDirErrorUnknownOperation)), nil)
	}
	return dispatcher.EndGroups
}
