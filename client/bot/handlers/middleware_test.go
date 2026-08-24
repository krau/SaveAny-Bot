package handlers

import (
	"testing"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
)

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
