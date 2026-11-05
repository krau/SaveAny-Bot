//go:build e2e

package telegrame2e

import (
	"context"
	"errors"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

// Dispatcher records handlers registered by client/bot/handlers and executes
// them in registration order, mirroring the production dispatcher groups.
type Dispatcher struct {
	handlers []dispatcher.Handler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) Initialize(context.Context, context.CancelFunc, *telegram.Client, *tg.User) {}

func (d *Dispatcher) Handle(context.Context, tg.UpdatesClass) error { return nil }

func (d *Dispatcher) AddHandler(handler dispatcher.Handler) {
	d.handlers = append(d.handlers, handler)
}

func (d *Dispatcher) AddHandlerToGroup(handler dispatcher.Handler, _ int) {
	d.handlers = append(d.handlers, handler)
}

func (d *Dispatcher) Dispatch(ctx *ext.Context, update *ext.Update) error {
	for _, handler := range d.handlers {
		err := handler.CheckUpdate(ctx, update)
		switch {
		case err == nil, errors.Is(err, dispatcher.ContinueGroups):
			continue
		case errors.Is(err, dispatcher.EndGroups):
			return nil
		case errors.Is(err, dispatcher.SkipCurrentGroup):
			return nil
		default:
			return err
		}
	}
	return nil
}
