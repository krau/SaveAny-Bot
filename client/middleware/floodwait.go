package middleware

import (
	"time"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"golang.org/x/time/rate"
)

func NewFloodWaitMiddlewares(maxRetries uint) []telegram.Middleware {
	waiter := floodwait.NewSimpleWaiter().WithMaxRetries(maxRetries)
	ratelimiter := ratelimit.New(rate.Every(time.Millisecond*100), 5)
	return []telegram.Middleware{
		waiter,
		ratelimiter,
	}
}
