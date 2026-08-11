package telegram

import (
	"context"
	"testing"

	"github.com/gotd/td/telegram/uploader"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
)

func TestUploadProgressAggregatesUploaderParts(t *testing.T) {
	type update struct {
		uploaded int64
		total    int64
	}
	var updates []update
	progress := newUploadProgress(100, func(uploaded, total int64) {
		updates = append(updates, update{uploaded: uploaded, total: total})
	})

	states := []uploader.ProgressState{
		{ID: 1, Uploaded: 20, Total: 60},
		{ID: 1, Uploaded: 20, Total: 60},
		{ID: 1, Uploaded: 60, Total: 60},
		{ID: 2, Uploaded: 10, Total: 40},
		{ID: 2, Uploaded: 40, Total: 40},
	}
	for _, state := range states {
		if err := progress.Chunk(context.Background(), state); err != nil {
			t.Fatalf("Chunk() failed: %v", err)
		}
	}

	want := []update{{20, 100}, {60, 100}, {70, 100}, {100, 100}}
	if len(updates) != len(want) {
		t.Fatalf("got %d updates, want %d", len(updates), len(want))
	}
	for i := range want {
		if updates[i] != want[i] {
			t.Fatalf("update %d = %+v, want %+v", i, updates[i], want[i])
		}
	}
}

func TestUploadProgressResetForSplitFiles(t *testing.T) {
	var uploaded, total int64
	progress := newUploadProgress(100, func(current, size int64) {
		uploaded, total = current, size
	})
	if err := progress.Chunk(context.Background(), uploader.ProgressState{ID: 1, Uploaded: 100, Total: 100}); err != nil {
		t.Fatalf("Chunk() failed: %v", err)
	}

	progress.reset(120)
	if err := progress.Chunk(context.Background(), uploader.ProgressState{ID: 2, Uploaded: 30, Total: 60}); err != nil {
		t.Fatalf("Chunk() after reset failed: %v", err)
	}
	if uploaded != 30 || total != 120 {
		t.Fatalf("progress after reset = %d/%d, want 30/120", uploaded, total)
	}
}

func TestBatchItemUploadProgressPreservesItemIndex(t *testing.T) {
	var gotIndex int
	var gotUploaded, gotTotal int64
	progress := batchItemUploadProgress(batchMediaItem{
		index: 4,
		item:  storagetypes.BatchItem{Size: 100},
	}, func(index int, uploaded, total int64) {
		gotIndex = index
		gotUploaded = uploaded
		gotTotal = total
	})
	if err := progress.Chunk(context.Background(), uploader.ProgressState{ID: 1, Uploaded: 25, Total: 100}); err != nil {
		t.Fatalf("Chunk() failed: %v", err)
	}
	if gotIndex != 4 || gotUploaded != 25 || gotTotal != 100 {
		t.Fatalf("batch progress = index %d, %d/%d; want index 4, 25/100", gotIndex, gotUploaded, gotTotal)
	}
}
