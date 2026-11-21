package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/selfupdate"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func TestHandleUpdateError(t *testing.T) {
	const detail = "failed to contact dav.internal.example"
	callback := &ext.Update{CallbackQuery: &tg.UpdateBotCallbackQuery{QueryID: 42}}
	for _, tt := range []struct {
		name        string
		update      *ext.Update
		message     string
		wantAlert   bool
		wantRestart bool
	}{
		{name: "callback failure", update: callback, message: detail, wantAlert: true},
		{name: "not a callback", update: &ext.Update{}, message: detail},
		{name: "background failure", message: detail},
		{name: "restart", update: callback, message: selfupdate.RestartSignal, wantRestart: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			restarted := false
			SetRestartHandler(func() { restarted = true })
			t.Cleanup(func() { SetRestartHandler(nil) })
			var answer *tg.MessagesSetBotCallbackAnswerRequest
			raw := tg.NewClient(telegram.InvokeFunc(func(_ context.Context, input bin.Encoder, output bin.Decoder) error {
				answer = input.(*tg.MessagesSetBotCallbackAnswerRequest)
				output.(*tg.BoolBox).Bool = &tg.BoolTrue{}
				return nil
			}))
			ctx := &ext.Context{Context: t.Context(), Raw: raw}
			if err := handleUpdateError(ctx, tt.update, tt.message); err != dispatcher.EndGroups {
				t.Fatalf("handleUpdateError() = %v, want EndGroups", err)
			}
			if restarted != tt.wantRestart {
				t.Errorf("restarted = %v, want %v", restarted, tt.wantRestart)
			}
			if (answer != nil) != tt.wantAlert {
				t.Fatalf("callback answer = %+v, want alert: %v", answer, tt.wantAlert)
			}
			if answer != nil && (answer.QueryID != 42 || !answer.Alert || answer.CacheTime != 0 ||
				answer.Message == "" || strings.Contains(answer.Message, "dav.internal.example")) {
				t.Errorf("callback answer = %+v, want an uncached, generic alert", answer)
			}
		})
	}
}
