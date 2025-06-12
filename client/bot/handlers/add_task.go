package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/cache"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleAddCallback(ctx *ext.Context, update *ext.Update) error {
	dataid := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	data, ok := cache.Get[tcbdata.Add](dataid)
	if !ok {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "数据已过期",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, update.CallbackQuery.GetUserID(), data.StorageName)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "存储获取失败: " + err.Error(),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	storagePath := selectedStorage.JoinStoragePath(data.File.Name())

	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	task, err := tftask.NewTGFileTask(injectCtx, data.File, ctx.Raw, selectedStorage, storagePath, tftask.NewProgressTrack(
		update.CallbackQuery.GetMsgID(),
		update.CallbackQuery.GetUserID()))
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "任务创建失败: " + err.Error(),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	if err := core.AddTask(injectCtx, dataid, task); err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "任务添加失败: " + err.Error(),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	length := core.GetLength(injectCtx)
	text := fmt.Sprintf("已添加到任务队列\n文件名: %s\n当前排队任务数: %d", data.File.Name(), length)
	if err := styling.Perform(&entityBuilder,
		styling.Plain("已添加到任务队列\n文件名: "),
		styling.Code(data.File.Name()),
		styling.Plain("\n当前排队任务数: "),
		styling.Bold(strconv.Itoa(length)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = entityBuilder.Complete()
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:       update.CallbackQuery.GetMsgID(),
		Message:  text,
		Entities: entities,
	})

	return dispatcher.EndGroups
}
