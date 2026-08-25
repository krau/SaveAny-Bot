package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
)

// Regression: callback queries usually arrive as updateShort without entity
// maps, so resolving the sender through the entity map yields ID 0 and every
// click was denied by the whitelist check. Callback updates must use the
// native UserID field.
func TestResponsibleUserID(t *testing.T) {
	tests := []struct {
		name   string
		update *ext.Update
		want   int64
	}{
		{
			name:   "callback query uses native user id",
			update: &ext.Update{CallbackQuery: &tg.UpdateBotCallbackQuery{UserID: 42}},
			want:   42,
		},
		{
			name: "message resolves through entity map",
			update: &ext.Update{
				EffectiveMessage: &types.Message{Message: &tg.Message{PeerID: &tg.PeerUser{UserID: 7}}},
				Entities:         &tg.Entities{Users: map[int64]*tg.User{7: {ID: 7}}},
			},
			want: 7,
		},
		{
			name: "callback query ignores entity map",
			update: &ext.Update{
				CallbackQuery: &tg.UpdateBotCallbackQuery{UserID: 9},
				Entities:      &tg.Entities{Users: map[int64]*tg.User{8: {ID: 8}}},
			},
			want: 9,
		},
		{
			name:   "unresolvable update yields zero",
			update: &ext.Update{},
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := responsibleUserID(tt.update); got != tt.want {
				t.Fatalf("responsibleUserID() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Regression: withPermission must treat ContinueGroups (the dispatcher's
// success sentinel) as a pass and invoke the wrapped handler. v0.60.1 treated
// it as an error, so every permitted callback was swallowed before the real
// handler ran.
func TestWithPermissionInvokesHandler(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("workers = 2\n\n[[users]]\nid = 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.Init(t.Context(), path); err != nil {
		t.Fatal(err)
	}

	update := &ext.Update{CallbackQuery: &tg.UpdateBotCallbackQuery{UserID: 42}}
	called := false
	handler := withPermission(func(ctx *ext.Context, u *ext.Update) error {
		called = true
		return nil
	})
	if err := handler(&ext.Context{}, update); err != nil {
		t.Fatalf("withPermission returned error: %v", err)
	}
	if !called {
		t.Fatal("withPermission did not invoke the wrapped handler")
	}
}
