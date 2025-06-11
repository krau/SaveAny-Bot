package bot

import (
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/core"
)

func cancelTask(ctx *ext.Context, update *ext.Update) error {
	key := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	err := core.CancelTask(ctx, key)
	if err == nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.QueryID,
			Message: "任务已取消",
		})
		return dispatcher.EndGroups
	}
	log.FromContext(ctx).Errorf("取消任务失败: %s", err)
	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.QueryID,
		Message: "任务取消失败",
	})
	return dispatcher.EndGroups
}
