package bot

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
)

func AddToQueue(ctx *ext.Context, update *ext.Update) error {
	if !slice.Contain(config.Cfg.GetUsersID(), update.CallbackQuery.UserID) {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "你没有权限",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	panic("Not implemented yet")
}
