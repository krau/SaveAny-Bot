package coreutil

import (
	"context"

	"github.com/krau/SaveAny-Bot/core"
)

func AddTask(ctx context.Context, task core.Exectable) error {
	// TODO: hook it so we can do something before adding the task (e.g. apply rules)
	return core.AddTask(ctx, task)
}
