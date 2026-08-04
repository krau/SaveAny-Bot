package batchtfile

import (
	"testing"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	tgstorage "github.com/krau/SaveAny-Bot/storage/telegram"
)

func TestExecutionGroupsPreserveSourceAlbums(t *testing.T) {
	stor := new(tgstorage.Telegram)
	otherStor := new(tgstorage.Telegram)
	task := Task{elems: []TaskElement{
		{Storage: stor, sourceGroupKey: "album-1"},
		{Storage: stor, sourceGroupKey: "album-1"},
		{Storage: stor},
		{Storage: stor, sourceGroupKey: "album-2"},
		{Storage: stor, sourceGroupKey: "album-2"},
		{Storage: otherStor, sourceGroupKey: "album-2"},
	}}

	groups := task.executionGroups()
	wantSizes := []int{2, 1, 2, 1}
	wantBatch := []bool{true, false, true, true}
	if len(groups) != len(wantSizes) {
		t.Fatalf("got %d groups, want %d", len(groups), len(wantSizes))
	}
	for i := range groups {
		if got := len(groups[i].elems); got != wantSizes[i] {
			t.Errorf("group %d has %d elements, want %d", i, got, wantSizes[i])
		}
		if got := groups[i].usesBatchSaver(); got != wantBatch[i] {
			t.Errorf("group %d batch=%v, want %v", i, got, wantBatch[i])
		}
	}
}

func TestSourceMetadataPreservesAlbumIdentityAndCaption(t *testing.T) {
	msg := &tg.Message{
		PeerID:  &tg.PeerChannel{ChannelID: 77},
		Message: "original caption",
	}
	msg.SetGroupedID(42)
	file := tfile.NewTGFile(nil, nil, 0, "photo.jpg", tfile.WithMessage(msg))

	groupKey, caption, preserveCaption := sourceMetadata(file)
	if groupKey != "*tg.PeerChannel:77:42" {
		t.Fatalf("group key = %q, want %q", groupKey, "*tg.PeerChannel:77:42")
	}
	if caption != "original caption" {
		t.Fatalf("caption = %q, want original caption", caption)
	}
	if !preserveCaption {
		t.Fatal("preserveCaption = false, want true")
	}
}
