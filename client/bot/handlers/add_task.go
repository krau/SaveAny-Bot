package handlers

import (
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleAddCallback(ctx *ext.Context, update *ext.Update) error {
	dataid := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	data, ok := cache.Get[tcbdata.Add](dataid)
	queryID := update.CallbackQuery.GetQueryID()
	if !ok {
		log.FromContext(ctx).Warnf("Invalid data ID: %s", dataid)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "数据已过期或无效"))
		return dispatcher.EndGroups
	}
	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, update.CallbackQuery.GetUserID(), data.StorageName)
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to get storage: %s", err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "存储获取失败: "+err.Error()))
		return dispatcher.EndGroups
	}

	storagePath := selectedStorage.JoinStoragePath(data.File.Name())

	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	task, err := tftask.NewTGFileTask(dataid, injectCtx, data.File, ctx.Raw, selectedStorage, storagePath, tftask.NewProgressTrack(
		update.CallbackQuery.GetMsgID(),
		update.CallbackQuery.GetUserID()))
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to create task: %s", err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "任务创建失败: "+err.Error()))
		return dispatcher.EndGroups
	}
	if err := core.AddTask(injectCtx, task); err != nil {
		log.FromContext(ctx).Errorf("Failed to add task: %s", err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "任务添加失败: "+err.Error()))
		return dispatcher.EndGroups
	}

	text, entities := msgelem.BuildTaskAddedEntities(ctx, data.File.Name(), core.GetLength(injectCtx))
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:       update.CallbackQuery.GetMsgID(),
		Message:  text,
		Entities: entities,
	})

	return dispatcher.EndGroups
}
