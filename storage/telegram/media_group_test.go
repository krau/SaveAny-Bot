package telegram

import (
	"bytes"
	"io"
	"testing"

	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
)

func TestPlanMediaGroups(t *testing.T) {
	tests := []struct {
		name      string
		items     []batchMediaItem
		wantSizes []int
	}{
		{
			name: "same source album",
			items: []batchMediaItem{
				albumItem("a", 1, true),
				albumItem("a", 1, true),
			},
			wantSizes: []int{2},
		},
		{
			name: "different source albums",
			items: []batchMediaItem{
				albumItem("a", 1, true),
				albumItem("b", 1, true),
			},
			wantSizes: []int{1, 1},
		},
		{
			name: "ungrouped messages",
			items: []batchMediaItem{
				albumItem("", 1, true),
				albumItem("", 1, true),
			},
			wantSizes: []int{1, 1},
		},
		{
			name: "different target chats",
			items: []batchMediaItem{
				albumItem("a", 1, true),
				albumItem("a", 2, true),
			},
			wantSizes: []int{1, 1},
		},
		{
			name: "ineligible media does not bridge albums",
			items: []batchMediaItem{
				albumItem("a", 1, true),
				albumItem("a", 1, false),
				albumItem("a", 1, true),
			},
			wantSizes: []int{1, 1, 1},
		},
		{
			name:      "maximum album size",
			items:     repeatedAlbumItems(11),
			wantSizes: []int{10, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := planMediaGroups(tt.items)
			if len(groups) != len(tt.wantSizes) {
				t.Fatalf("got %d groups, want %d", len(groups), len(tt.wantSizes))
			}
			for i, want := range tt.wantSizes {
				if got := len(groups[i]); got != want {
					t.Errorf("group %d has %d items, want %d", i, got, want)
				}
			}
		})
	}
}

func TestMediaCaption(t *testing.T) {
	empty := ""
	original := "original caption"
	tests := []struct {
		name     string
		override *string
		wantLen  int
	}{
		{name: "filename fallback", wantLen: 1},
		{name: "preserve empty source caption", override: &empty, wantLen: 0},
		{name: "preserve source caption", override: &original, wantLen: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(mediaCaption("file.jpg", tt.override)); got != tt.wantLen {
				t.Fatalf("got %d caption options, want %d", got, tt.wantLen)
			}
		})
	}
}

func TestInspectBatchItemRewindsBeforeMimetypeDetection(t *testing.T) {
	data := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	reader := bytes.NewReader(data)
	if _, err := reader.Seek(4, io.SeekStart); err != nil {
		t.Fatalf("failed to set initial reader offset: %v", err)
	}

	mediaItem, err := new(Telegram).inspectBatchItem(nil, storagetypes.BatchItem{
		Reader:      reader,
		StoragePath: "photo.png",
		Size:        int64(len(data)),
	})
	if err != nil {
		t.Fatalf("inspectBatchItem returned an error: %v", err)
	}
	if !mediaItem.albumEligible {
		t.Fatal("albumEligible = false, want true for PNG input")
	}
	if offset, err := reader.Seek(0, io.SeekCurrent); err != nil {
		t.Fatalf("failed to get final reader offset: %v", err)
	} else if offset != 0 {
		t.Fatalf("reader offset = %d, want 0", offset)
	}
}

func albumItem(group string, chatID int64, eligible bool) batchMediaItem {
	return batchMediaItem{
		item:          storagetypes.BatchItem{SourceGroupKey: group},
		chatID:        chatID,
		albumEligible: eligible,
	}
}

func repeatedAlbumItems(count int) []batchMediaItem {
	items := make([]batchMediaItem, count)
	for i := range items {
		items[i] = albumItem("a", 1, true)
	}
	return items
}
