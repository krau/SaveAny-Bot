package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
	tgstorage "github.com/krau/mygotg/storage"
	"github.com/krau/mygotg/types"
)

func TestDirMenuWithoutDirs(t *testing.T) {
	const userID = 42
	database.Init(t.Context(), filepath.Join(t.TempDir(), "test.db"))
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := database.CreateUser(t.Context(), userID); err != nil {
		t.Fatal(err)
	}

	peer := &tg.PeerUser{UserID: userID}
	peers := tgstorage.NewPeerStorage(nil, true)
	peers.AddPeer(userID, 1, tgstorage.TypeUser, "")
	entities := &tg.Entities{Users: map[int64]*tg.User{userID: {ID: userID, AccessHash: 1}}}
	var sent *tg.MessagesSendMessageRequest
	var edited *tg.MessagesEditMessageRequest
	raw := tg.NewClient(telegram.InvokeFunc(func(_ context.Context, input bin.Encoder, output bin.Decoder) error {
		if err := input.Encode(new(bin.Buffer)); err != nil {
			t.Errorf("encode %T: %v", input, err)
			return err
		}
		switch req := input.(type) {
		case *tg.MessagesSendMessageRequest:
			sent = req
			output.(*tg.UpdatesBox).Updates = &tg.UpdateShortSentMessage{ID: 2}
		case *tg.MessagesEditMessageRequest:
			edited = req
			output.(*tg.UpdatesBox).Updates = &tg.UpdateShort{Update: &tg.UpdateEditMessage{
				Message: &tg.Message{PeerID: peer, ID: req.ID, Message: req.Message},
			}}
		default:
			return fmt.Errorf("unexpected request %T", input)
		}
		return nil
	}))
	ctx := ext.NewContext(t.Context(), raw, peers, nil, message.NewSender(raw), entities, false)
	msg := &tg.Message{ID: 1, PeerID: peer, Message: "/dirs"}
	msg.SetFromID(peer)
	if err := handleDirCmd(ctx, &ext.Update{
		UpdateClass:      &tg.UpdateNewMessage{Message: msg},
		EffectiveMessage: types.ConstructMessage(msg),
		Entities:         entities,
	}); err != dispatcher.EndGroups {
		t.Fatalf("handleDirCmd: %v", err)
	}
	if sent == nil || sent.ReplyMarkup != nil || sent.Message != dirMenuText(nil) {
		t.Errorf("empty /dirs request = %+v; want the empty menu text without markup", sent)
	}

	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CreateDirForUser(ctx, user.ID, 1, "album"); err != nil {
		t.Fatal(err)
	}
	dirs, err := database.GetUserDirs(ctx, user.ID)
	if err != nil || len(dirs) != 1 {
		t.Fatalf("GetUserDirs() = %v, %v; want one directory", dirs, err)
	}
	if err := handleDirCallback(ctx, &ext.Update{CallbackQuery: &tg.UpdateBotCallbackQuery{
		UserID: userID, Peer: peer, MsgID: 2,
		Data: fmt.Appendf(nil, "%s del %d", tcbdata.TypeDir, dirs[0].ID),
	}}); err != dispatcher.EndGroups {
		t.Fatalf("handleDirCallback: %v", err)
	}
	if edited == nil || edited.Message != dirMenuText(nil) {
		t.Fatalf("edit request after deleting last dir = %+v; want the empty menu text", edited)
	}
	if markup, ok := edited.ReplyMarkup.(*tg.ReplyInlineMarkup); !ok || len(markup.Rows) != 0 {
		t.Errorf("edit markup = %+v; want an empty inline keyboard", edited.ReplyMarkup)
	}
}
