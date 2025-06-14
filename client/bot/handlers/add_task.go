package handlers

import (
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleAddOneCallback(ctx *ext.Context, update *ext.Update) error {
	dataid := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	data, err := shortcut.GetCallbackDataWithAnswer[tcbdata.Add](ctx, update, dataid)
	if err != nil {
		return err
	}
	queryID := update.CallbackQuery.GetQueryID()
	userID := update.CallbackQuery.GetUserID()
	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, data.StorageName)
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to get storage: %s", err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "存储获取失败: "+err.Error()))
		return dispatcher.EndGroups
	}
	return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, selectedStorage, data.File, userID, update.CallbackQuery.GetMsgID())
}

func handleAddBatchCallback(ctx *ext.Context, update *ext.Update) error {
	dataid := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	queryID := update.CallbackQuery.GetQueryID()
	data, err := shortcut.GetCallbackDataWithAnswer[tcbdata.AddBatch](ctx, update, dataid)
	if err != nil {
		return err
	}
	userID := update.CallbackQuery.GetUserID()
	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, data.SelectedStorage)
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to get storage: %s", err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "存储获取失败: "+err.Error()))
		return dispatcher.EndGroups
	}
	trackMsgID := update.CallbackQuery.GetMsgID()
	return shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, selectedStorage, data.Files, userID, trackMsgID)
}
