package bot

import (
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/krau/SaveAny-Bot/config"
	"golang.org/x/time/rate"
)

func FloodWaitMiddleware() []telegram.Middleware {
	waiter := floodwait.NewSimpleWaiter().WithMaxRetries(uint(config.Cfg.Telegram.FloodRetry))
	ratelimiter := ratelimit.New(rate.Every(time.Millisecond*100), 5)
	return []telegram.Middleware{
		waiter,
		ratelimiter,
	}
}

const noPermissionText string = `
您不在白名单中, 无法使用此 Bot.
您可以部署自己的实例: https://github.com/krau/SaveAny-Bot
`

func checkPermission(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	if !slice.Contain(config.Cfg.GetUsersID(), userID) {
		ctx.Reply(update, ext.ReplyTextString(noPermissionText), nil)
		return dispatcher.EndGroups
	}
	return dispatcher.ContinueGroups
}
