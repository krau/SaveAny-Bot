package tdler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"

	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

// failAfterClient serves the first failAfter chunks, then returns err.
type failAfterClient struct {
	*serverLikeClient
	failAfter int
	calls     int
	err       error
}

func (c *failAfterClient) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	c.calls++
	if c.calls > c.failAfter {
		return nil, c.err
	}
	return c.serverLikeClient.UploadGetFile(ctx, req)
}

func TestDownloadResumableFull(t *testing.T) {
	data := make([]byte, 3*1024*1024+123)
	for i := range data {
		data[i] = byte(i % 251)
	}
	client := &serverLikeClient{data: data}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "test.bin")

	dir := t.TempDir()
	bitmapPath := filepath.Join(dir, "test.bin.bitmap")
	w := &memWriterAt{b: make([]byte, len(data))}

	if err := DownloadResumable(context.Background(), file, w, 4, bitmapPath); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if !bytesEqual(w.b, data) {
		t.Fatalf("downloaded data mismatch")
	}
	bm, err := loadResumeBitmap(bitmapPath)
	if err != nil {
		t.Fatalf("load bitmap: %v", err)
	}
	if bm == nil || !bm.complete() {
		t.Fatalf("bitmap not complete after full download")
	}
}

func TestDownloadResumableInterrupted(t *testing.T) {
	data := make([]byte, 5*1024*1024) // exactly 5 blocks
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	bitmapPath := filepath.Join(dir, "test.bin.bitmap")
	w := &memWriterAt{b: make([]byte, len(data))}

	// First run: 3 blocks complete, 4th request fails.
	flaky := &failAfterClient{
		serverLikeClient: &serverLikeClient{data: data},
		failAfter:        3,
		err:              tgerr.New(500, "INTERNAL_SERVER_ERROR"),
	}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, flaky, int64(len(data)), "test.bin")
	err := DownloadResumable(context.Background(), file, w, 1, bitmapPath)
	if err == nil {
		t.Fatalf("expected first run to fail")
	}
	bm, err := loadResumeBitmap(bitmapPath)
	if err != nil {
		t.Fatalf("load bitmap after interruption: %v", err)
	}
	if bm == nil {
		t.Fatalf("bitmap missing after interruption")
	}
	if got := bm.blockCount() - len(bm.missingBlocks()); got != 3 {
		t.Fatalf("expected 3 completed blocks, got %d", got)
	}

	// Second run: only the missing blocks are requested.
	healthy := &serverLikeClient{data: data}
	file = tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, healthy, int64(len(data)), "test.bin")
	if err := DownloadResumable(context.Background(), file, w, 1, bitmapPath); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if !bytesEqual(w.b, data) {
		t.Fatalf("resumed data mismatch")
	}
	if healthy.maxOffset >= int64(len(data)) {
		t.Fatalf("resume requested offset %d at or past EOF", healthy.maxOffset)
	}
	if bm, err = loadResumeBitmap(bitmapPath); err != nil || bm == nil || !bm.complete() {
		t.Fatalf("bitmap not complete after resume: %v", err)
	}
}

func TestDownloadResumableBitmapResetOnSizeChange(t *testing.T) {
	data := make([]byte, 2*1024*1024)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	bitmapPath := filepath.Join(dir, "test.bin.bitmap")
	w := &memWriterAt{b: make([]byte, len(data))}

	// Record a bitmap claiming the old, larger file is fully downloaded.
	stale := newResumeBitmap(int64(4 * 1024 * 1024))
	if err := stale.save(bitmapPath); err != nil {
		t.Fatalf("save stale bitmap: %v", err)
	}

	client := &serverLikeClient{data: data}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "test.bin")
	if err := DownloadResumable(context.Background(), file, w, 1, bitmapPath); err != nil {
		t.Fatalf("download with stale bitmap failed: %v", err)
	}
	if !bytesEqual(w.b, data) {
		t.Fatalf("data mismatch with stale bitmap")
	}
}

func TestRemoveResumeState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.bitmap")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveResumeState(path); err != nil {
		t.Fatalf("RemoveResumeState: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bitmap still exists: %v", err)
	}
	// Removing again must be a no-op.
	if err := RemoveResumeState(path); err != nil {
		t.Fatalf("RemoveResumeState second call: %v", err)
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
