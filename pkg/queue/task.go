package queue

import (
	"container/list"
	"context"
	"time"
)

type Task[T any] struct {
	ID      string
	Data    T
	ctx     context.Context
	cancel  context.CancelFunc
	created time.Time
	element *list.Element
}

func NewTask[T any](ctx context.Context, id string, data T) *Task[T] {
	cancelCtx, cancel := context.WithCancel(ctx)
	return &Task[T]{
		ID:      id,
		Data:    data,
		ctx:     cancelCtx,
		cancel:  cancel,
		created: time.Now(),
	}
}

func (t *Task[T]) IsCancelled() bool {
	select {
	case <-t.ctx.Done():
		return true
	default:
		return false
	}
}

func (t *Task[T]) Cancel() {
	t.cancel()
}

func (t *Task[T]) Context() context.Context {
	return t.ctx
}
