package middleware

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/telegram"
	"github.com/krau/SaveAny-Bot/client/middleware/recovery"
	"github.com/krau/SaveAny-Bot/client/middleware/retry"
	"github.com/krau/SaveAny-Bot/config"
)

// https://github.com/iyear/tdl/blob/master/core/tclient/tclient.go
func NewDefaultMiddlewares(ctx context.Context, timeout time.Duration) []telegram.Middleware {
	return []telegram.Middleware{
		recovery.New(ctx, newBackoff(timeout)),
		retry.New(config.C().Telegram.RpcRetry),
		floodwait.NewSimpleWaiter(),
	}
}

func newBackoff(timeout time.Duration) backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.Multiplier = 1.1
	b.MaxElapsedTime = timeout
	b.MaxInterval = 10 * time.Second
	return b
}
