package bot

import (
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/queue"
)

func cancelTask(ctx *ext.Context, update *ext.Update) error {
	key := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	ok := queue.CancelTask(key)
	if ok {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.QueryID,
			Message: "任务已取消",
		})
		return dispatcher.EndGroups
	}
	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.QueryID,
		Message: "任务取消失败",
	})
	return dispatcher.EndGroups
}
