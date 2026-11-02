//go:build e2e

package telegrame2e

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gotd/td/tg"
)

var harness *Harness

func TestMain(m *testing.M) {
	var err error
	harness, err = NewHarness()
	if err != nil {
		fmt.Fprintln(os.Stderr, "setup telegram harness:", err)
		os.Exit(1)
	}
	code := m.Run()
	harness.Close()
	os.Exit(code)
}

func TestBotStartCommand(t *testing.T) {
	harness.Server.ResetRecords()
	peer := harness.UserPeer(localUserID)
	message := harness.Server.AddTextMessage(peer, peer, "/start")
	if err := harness.Dispatch(harness.MessageUpdate(message)); err != nil {
		t.Fatalf("dispatch /start: %v", err)
	}
	sent := harness.Server.Sent()
	if len(sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sent))
	}
	if !strings.Contains(sent[0].Text, "Save Any Bot") {
		t.Fatalf("help text = %q", sent[0].Text)
	}
}

func TestGroupGate(t *testing.T) {
	group := &tg.PeerChat{ChatID: groupID}
	from := harness.UserPeer(localUserID)

	harness.Server.ResetRecords()
	media := harness.Server.AddDocumentMessage(group, from, "group.txt", []byte("group"))
	if err := harness.Dispatch(harness.MessageUpdate(media)); err != nil {
		t.Fatalf("dispatch group media: %v", err)
	}
	if sent := harness.Server.Sent(); len(sent) != 0 {
		t.Fatalf("plain group media produced %d messages", len(sent))
	}

	command := harness.Server.AddTextMessage(group, from, "/start@testbot")
	if err := harness.Dispatch(harness.MessageUpdate(command)); err != nil {
		t.Fatalf("dispatch group command: %v", err)
	}
	sent := harness.Server.Sent()
	if len(sent) != 1 {
		t.Fatalf("group command sent %d messages, want 1", len(sent))
	}
	if _, ok := sent[0].Peer.(*tg.PeerChat); !ok {
		t.Fatalf("group command peer = %T, want PeerChat", sent[0].Peer)
	}

	harness.Server.ResetRecords()
	stranger := harness.UserPeer(10003)
	unauthorized := harness.Server.AddTextMessage(group, stranger, "/start@testbot")
	if err := harness.Dispatch(harness.MessageUpdate(unauthorized)); err != nil {
		t.Fatalf("dispatch unauthorized: %v", err)
	}
	if sent := harness.Server.Sent(); len(sent) != 0 {
		t.Fatalf("unauthorized group command produced %d messages", len(sent))
	}

	harness.Server.ResetRecords()
	channel := &tg.PeerChannel{ChannelID: channelID}
	anonymous := harness.Server.AddDocumentMessage(channel, channel, "anon.txt", []byte("anon"))
	if err := harness.Dispatch(harness.MessageUpdate(anonymous)); err != nil {
		t.Fatalf("dispatch anonymous admin: %v", err)
	}
	if sent := harness.Server.Sent(); len(sent) != 0 {
		t.Fatalf("anonymous admin produced %d messages", len(sent))
	}
}

func TestSaveTelegramFileToLocal(t *testing.T) {
	harness.Server.ResetRecords()
	if err := harness.ClearLocal(); err != nil {
		t.Fatalf("clear local storage: %v", err)
	}
	peer := harness.UserPeer(localUserID)
	content := []byte("hello from telegram\n")
	document := harness.Server.AddDocumentMessage(peer, peer, "hello.txt", content)
	command := harness.Server.AddTextMessage(peer, peer, "/save")
	command.SetReplyTo(&tg.MessageReplyHeader{ReplyToMsgID: document.ID})

	update := harness.CommandUpdate(command, document)
	if err := harness.Dispatch(update); err != nil {
		t.Fatalf("dispatch /save: %v", err)
	}
	got, err := harness.WaitLocalFile(20 * time.Second)
	if err != nil {
		t.Fatalf("wait for saved file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("saved content = %q, want %q", got, content)
	}
	if edits := harness.Server.Edits(); len(edits) == 0 {
		t.Fatal("task produced no progress edits")
	}
}

func TestSaveTelegramFileToTelegramStorage(t *testing.T) {
	harness.Server.ResetRecords()
	peer := harness.UserPeer(tgUserID)
	content := []byte("uploaded to telegram storage\n")
	document := harness.Server.AddDocumentMessage(peer, peer, "tg-upload.txt", content)
	command := harness.Server.AddTextMessage(peer, peer, "/save")
	command.SetReplyTo(&tg.MessageReplyHeader{ReplyToMsgID: document.ID})

	update := harness.CommandUpdate(command, document)
	if err := harness.Dispatch(update); err != nil {
		t.Fatalf("dispatch /save: %v", err)
	}
	got, err := harness.WaitUploaded(20 * time.Second)
	if err != nil {
		t.Fatalf("wait for telegram upload: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("uploaded content = %q, want %q", got, content)
	}
	if sent := harness.Server.Sent(); len(sent) == 0 {
		t.Fatal("telegram storage did not send a media message")
	}
}
