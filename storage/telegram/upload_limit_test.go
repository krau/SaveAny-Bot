package telegram

import (
	"bytes"
	"testing"

	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
)

func TestMaxUploadFileSize(t *testing.T) {
	tests := []struct {
		name string
		ctx  *ext.Context
		want int64
	}{
		{name: "missing context", want: MaxUploadFileSize},
		{name: "missing self", ctx: new(ext.Context), want: MaxUploadFileSize},
		{
			name: "bot",
			ctx:  uploadAccountContext(true, false),
			want: MaxUploadFileSize,
		},
		{
			name: "premium bot still uses bot limit",
			ctx:  uploadAccountContext(true, true),
			want: MaxUploadFileSize,
		},
		{
			name: "regular user",
			ctx:  uploadAccountContext(false, false),
			want: MaxUploadFileSize,
		},
		{
			name: "premium user",
			ctx:  uploadAccountContext(false, true),
			want: PremiumMaxUploadFileSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maxUploadFileSize(tt.ctx); got != tt.want {
				t.Fatalf("maxUploadFileSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func uploadAccountContext(bot, premium bool) *ext.Context {
	self := new(tg.User)
	self.SetBot(bot)
	self.SetPremium(premium)
	return &ext.Context{Self: self}
}

func TestSplitSizeUsesUploaderLimit(t *testing.T) {
	tests := []struct {
		name         string
		splitSizeMB  int64
		accountLimit int64
		want         int64
	}{
		{
			name:         "bot default",
			accountLimit: MaxUploadFileSize,
			want:         MaxUploadFileSize,
		},
		{
			name:         "premium default",
			accountLimit: PremiumMaxUploadFileSize,
			want:         PremiumMaxUploadFileSize,
		},
		{
			name:         "explicit lower limit",
			splitSizeMB:  1500,
			accountLimit: PremiumMaxUploadFileSize,
			want:         1500 * 1024 * 1024,
		},
		{
			name:         "explicit limit is capped by account",
			splitSizeMB:  5000,
			accountLimit: PremiumMaxUploadFileSize,
			want:         PremiumMaxUploadFileSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telegramStorage := Telegram{}
			telegramStorage.config.SplitSizeMB = tt.splitSizeMB
			if got := telegramStorage.splitSize(tt.accountLimit); got != tt.want {
				t.Fatalf("splitSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBatchUploadLimitAppliesPerFile(t *testing.T) {
	telegramStorage := Telegram{}
	tctx := uploadAccountContext(true, false)
	data := []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom")
	itemSize := int64(MaxUploadFileSize/2 + 1)

	for index := range 2 {
		item, err := telegramStorage.inspectBatchItem(tctx, storagetypes.BatchItem{
			Reader:      bytes.NewReader(data),
			StoragePath: "video.mp4",
			Size:        itemSize,
		})
		if err != nil {
			t.Fatalf("inspectBatchItem() failed for item %d: %v", index, err)
		}
		if item.useSingleSave {
			t.Fatalf("item %d was treated as oversized even though only the batch total exceeds the limit", index)
		}
	}
}
