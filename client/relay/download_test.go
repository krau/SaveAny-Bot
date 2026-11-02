package relay

import (
	"context"
	"sync"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/worker"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/mygotg/ext"
	tgstorage "github.com/krau/mygotg/storage"
)

type relayFakeInvoker struct {
	mu       sync.Mutex
	forwards int
	deletes  int
	relayMsg *tg.Message
	relayDoc *tg.Document
}

func (f *relayFakeInvoker) Invoke(_ context.Context, input bin.Encoder, output bin.Decoder) error {
	switch req := input.(type) {
	case *tg.MessagesForwardMessagesRequest:
		f.mu.Lock()
		f.forwards++
		f.mu.Unlock()
		if box, ok := output.(*tg.UpdatesBox); ok {
			box.Updates = &tg.Updates{Updates: []tg.UpdateClass{&tg.UpdateNewChannelMessage{Message: f.relayMsg}}}
		}
	case *tg.ChannelsGetMessagesRequest:
		if box, ok := output.(*tg.MessagesMessagesBox); ok {
			box.Messages = &tg.MessagesChannelMessages{Messages: []tg.MessageClass{f.relayMsg}}
		}
	case *tg.ChannelsDeleteMessagesRequest:
		f.mu.Lock()
		f.deletes++
		f.mu.Unlock()
	case *tg.UploadGetFileRequest:
		if box, ok := output.(*tg.UploadFileBox); ok {
			box.File = &tg.UploadFile{Bytes: make([]byte, req.Limit)}
		}
	}
	return nil
}

func relayTestExt(inv *relayFakeInvoker, peers []int64) *ext.Context {
	storage := tgstorage.NewPeerStorage(nil, true)
	for _, id := range peers {
		storage.AddPeer(id, 1, tgstorage.TypeChannel, "")
	}
	return ext.NewContext(context.Background(), tg.NewClient(inv), storage, &tg.User{ID: 1}, nil, nil, false)
}

func TestPrepareResumableReusesRelayMessage(t *testing.T) {
	oldCfg := config.C()
	cfg := oldCfg
	cfg.Telegram.RelayChatID = 500
	cfg.Telegram.RelayKeepMessages = false
	config.Set(cfg)
	defer config.Set(oldCfg)

	const sourceChatID int64 = 400
	doc := &tg.Document{ID: 7, AccessHash: 8, FileReference: []byte{1}, Size: 9 << 20, DCID: 2}
	relayMsg := &tg.Message{ID: 777, PeerID: &tg.PeerChannel{ChannelID: 500}, Media: &tg.MessageMediaDocument{Document: doc}}
	sourceMsg := &tg.Message{ID: 10, PeerID: &tg.PeerChannel{ChannelID: sourceChatID}, Media: &tg.MessageMediaDocument{Document: doc}}
	primaryInv := &relayFakeInvoker{relayMsg: relayMsg, relayDoc: doc}
	primary := relayTestExt(primaryInv, []int64{sourceChatID, 500})
	worker1 := &relayFakeInvoker{relayMsg: relayMsg, relayDoc: doc}
	worker2 := &relayFakeInvoker{relayMsg: relayMsg, relayDoc: doc}
	oldRunning := runningClients
	runningClients = func() []worker.ClientRef {
		return []worker.ClientRef{
			{ID: 1, Name: "one", Ext: relayTestExt(worker1, []int64{500})},
			{ID: 2, Name: "two", Ext: relayTestExt(worker2, []int64{500})},
		}
	}
	defer func() { runningClients = oldRunning }()

	file, err := tfile.FromMediaMessage(sourceMsg.Media, tg.NewClient(primaryInv), sourceMsg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tgutil.ExtWithContext(context.Background(), primary)
	statePath := ResumeStatePath(t.TempDir() + "/cache")

	first, firstCleanup := PrepareResumable(ctx, file, statePath)
	if first == nil || firstCleanup == nil {
		t.Fatal("relay was not prepared")
	}
	defer firstCleanup(ctx)
	primaryInv.mu.Lock()
	forwards := primaryInv.forwards
	primaryInv.mu.Unlock()
	if forwards != 1 {
		t.Fatalf("forward count = %d, want 1", forwards)
	}
	if _, err := loadResumeState(statePath); err != nil {
		t.Fatalf("state not persisted: %v", err)
	}

	second, cleanup2 := PrepareResumable(ctx, file, statePath)
	if second == nil || cleanup2 == nil {
		t.Fatal("relay state was not reused")
	}
	primaryInv.mu.Lock()
	forwards = primaryInv.forwards
	primaryInv.mu.Unlock()
	if forwards != 1 {
		t.Fatalf("second prepare forwarded again: %d", forwards)
	}
	cleanup2(ctx)
	if state, err := loadResumeState(statePath); err != nil || state != nil {
		t.Fatalf("relay state after cleanup = %+v, %v", state, err)
	}
	primaryInv.mu.Lock()
	deletes := primaryInv.deletes
	primaryInv.mu.Unlock()
	if deletes == 0 {
		t.Fatal("relay message was not deleted")
	}
}
