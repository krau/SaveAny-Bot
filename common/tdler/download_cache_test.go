package tdler

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"

	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

// TestDownloadToCacheCompleteCleansOrphanBitmap verifies a leftover bitmap
// from a crashed run (rename done, cleanup not) is removed when the complete
// cache file is reused.
func TestDownloadToCacheCompleteCleansOrphanBitmap(t *testing.T) {
	data := make([]byte, 1024*1024+7)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.bin")
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	bitmapPath := ResumeStatePath(cachePath + ".part")
	if err := os.WriteFile(bitmapPath, []byte(`{"part_size":1048576,"size":1048583,"blocks":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	client := &countingClient{serverLikeClient: &serverLikeClient{data: data}, onCall: func() { t.Fatalf("unexpected download request") }}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "cache.bin")
	if err := DownloadToCache(context.Background(), file, cachePath, func(w io.WriterAt) io.WriterAt { return w }); err != nil {
		t.Fatalf("DownloadToCache: %v", err)
	}
	if _, err := os.Stat(bitmapPath); !os.IsNotExist(err) {
		t.Fatalf("orphan bitmap still exists: %v", err)
	}
}

// TestDownloadToCacheResume verifies an interrupted .part download resumes
// and finalizes: cache file created, bitmap removed, data complete.
func TestDownloadToCacheResume(t *testing.T) {
	data := make([]byte, 5*1024*1024)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.bin")
	partPath := cachePath + ".part"
	bitmapPath := ResumeStatePath(partPath)

	// First run: 3 blocks complete, then the server fails.
	flaky := &failAfterClient{
		serverLikeClient: &serverLikeClient{data: data},
		failAfter:        3,
		err:              tgerr.New(500, "INTERNAL_SERVER_ERROR"),
	}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, flaky, int64(len(data)), "cache.bin")
	if err := DownloadToCache(context.Background(), file, cachePath, func(w io.WriterAt) io.WriterAt { return w }); err == nil {
		t.Fatalf("expected first run to fail")
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache file must not exist after interrupted download")
	}

	// Second run: resumes and finalizes.
	healthy := &serverLikeClient{data: data}
	file = tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, healthy, int64(len(data)), "cache.bin")
	if err := DownloadToCache(context.Background(), file, cachePath, func(w io.WriterAt) io.WriterAt { return w }); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	got, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, data) {
		t.Fatalf("cache content mismatch")
	}
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Fatalf(".part file still exists after finalize")
	}
	if _, err := os.Stat(bitmapPath); !os.IsNotExist(err) {
		t.Fatalf("bitmap still exists after finalize")
	}
}

// TestDownloadToCacheUnknownSize verifies files without a known size (e.g.
// photos) go through the plain downloader and land directly in the cache.
func TestDownloadToCacheUnknownSize(t *testing.T) {
	data := make([]byte, 1024*1024+13)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "photo.bin")

	client := &serverLikeClient{data: data}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, 0, "photo.bin")
	if err := DownloadToCache(context.Background(), file, cachePath, func(w io.WriterAt) io.WriterAt { return w }); err != nil {
		t.Fatalf("DownloadToCache: %v", err)
	}
	got, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, data) {
		t.Fatalf("cache content mismatch")
	}
	if _, err := os.Stat(cachePath + ".part"); !os.IsNotExist(err) {
		t.Fatalf("unexpected .part file for unknown-size download")
	}
}

// countingClient counts upload.getFile requests.
type countingClient struct {
	*serverLikeClient
	onCall func()
}

func (c *countingClient) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	c.onCall()
	return c.serverLikeClient.UploadGetFile(ctx, req)
}
