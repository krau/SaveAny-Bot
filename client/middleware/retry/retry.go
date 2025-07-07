package retry

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

var internalErrors = []string{
	"Timedout", // #373
	"No workers running",
	"RPC_CALL_FAIL",
	"RPC_MCGET_FAIL",
	"WORKER_BUSY_TOO_LONG_RETRY", // #462
	"memory limit exit",          // #504
}

type retry struct {
	max    int
	errors []string
}

func (r retry) Handle(next tg.Invoker) telegram.InvokeFunc {
	return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		retries := 0

		for retries < r.max {
			if err := next.Invoke(ctx, input, output); err != nil {
				if tgerr.Is(err, r.errors...) {
					log.FromContext(ctx).Debug("retry middleware", "retries", retries, "error", err)
					retries++
					continue
				}
				// retry middleware skip
				return err
			}

			return nil
		}

		return fmt.Errorf("retry limit reached after %d attempts", r.max)
	}
}

// New returns middleware that retries request if it fails with one of provided errors.
func New(max int, errors ...string) telegram.Middleware {
	return retry{
		max:    max,
		errors: append(errors, internalErrors...), // #373
	}
}
