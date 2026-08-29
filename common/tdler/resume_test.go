package tdler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// TestDownloadResumablePartMissingOrTruncated resets the bitmap: skipped
// blocks would otherwise be zero-filled (caller recreates the part file
// without its bytes), or the download would wedge forever on a stale
// complete bitmap.
func TestDownloadResumablePartMissingOrTruncated(t *testing.T) {
	data := make([]byte, 5*1024*1024)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	partPath := filepath.Join(dir, "test.bin.part")
	bitmapPath := ResumeStatePath(partPath)

	tests := []struct {
		name       string
		doneBlocks []int
		createPart bool
		truncate   bool
	}{
		{"part missing, partial bitmap", []int{0, 1, 2}, false, false},
		{"part empty, partial bitmap", []int{0, 1, 2}, true, true},
		{"part missing, complete bitmap", []int{0, 1, 2, 3, 4}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Remove(partPath)
			os.Remove(bitmapPath)
			bm := newResumeBitmap(int64(len(data)))
			for _, block := range tt.doneBlocks {
				bm.markDone(block)
			}
			if err := bm.save(bitmapPath); err != nil {
				t.Fatal(err)
			}
			if tt.createPart {
				// Simulate the caller re-creating the part file (truncating).
				if err := os.WriteFile(partPath, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				if tt.truncate {
					if err := os.WriteFile(partPath, make([]byte, 0), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}

			partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			defer partFile.Close()
			client := &serverLikeClient{data: data}
			file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "test.bin")
			if err := DownloadResumable(context.Background(), file, partFile, 1, bitmapPath); err != nil {
				t.Fatalf("download failed: %v", err)
			}
			got := make([]byte, len(data))
			if _, err := partFile.ReadAt(got, 0); err != nil {
				t.Fatal(err)
			}
			if !bytesEqual(got, data) {
				t.Fatalf("downloaded data mismatch (blocks not reset)")
			}
		})
	}
}

// TestDownloadResumableInvalidBitmap treats a corrupt bitmap as absent.
func TestDownloadResumableInvalidBitmap(t *testing.T) {
	data := make([]byte, 1024*1024+7)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	partPath := filepath.Join(dir, "test.bin.part")
	bitmapPath := ResumeStatePath(partPath)
	for _, content := range []string{
		`{"part_size":1048576,"size":-1,"blocks":[]}`,
		`{"part_size":1048576,"size":9223372036854775807,"blocks":[]}`,
		`not json`,
	} {
		os.Remove(partPath)
		if err := os.WriteFile(bitmapPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		client := &serverLikeClient{data: data}
		file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "test.bin")
		err = DownloadResumable(context.Background(), file, partFile, 1, bitmapPath)
		partFile.Close()
		if err != nil {
			t.Fatalf("download with corrupt bitmap %q failed: %v", content, err)
		}
		got := make([]byte, len(data))
		f, err := os.Open(partPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.ReadAt(got, 0); err != nil {
			t.Fatal(err)
		}
		f.Close()
		if !bytesEqual(got, data) {
			t.Fatalf("downloaded data mismatch with corrupt bitmap %q", content)
		}
	}
}

// cdnRedirectClient answers the first redirects requests with a CDN
// redirect, then serves chunks normally (as a CDN-free fallback would).
type cdnRedirectClient struct {
	*serverLikeClient
	redirects int
}

func (c *cdnRedirectClient) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	if c.redirects > 0 {
		c.redirects--
		return &tg.UploadFileCDNRedirect{
			DCID:          2,
			FileToken:     []byte("token"),
			EncryptionKey: make([]byte, 32),
			EncryptionIv:  make([]byte, 16),
		}, nil
	}
	return c.serverLikeClient.UploadGetFile(ctx, req)
}

// TestDownloadResumableCDNFallback verifies that a CDN redirect aborts the
// raw chunked download and falls back to the plain gotd downloader, which
// handles CDN re-hashing internally.
func TestDownloadResumableCDNFallback(t *testing.T) {
	data := make([]byte, 3*1024*1024+123)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	bitmapPath := filepath.Join(dir, "test.bin.bitmap")
	w := &memWriterAt{b: make([]byte, len(data))}

	client := &cdnRedirectClient{serverLikeClient: &serverLikeClient{data: data}, redirects: 1}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "test.bin")
	if err := DownloadResumable(context.Background(), file, w, 1, bitmapPath); err != nil {
		t.Fatalf("download with CDN redirect failed: %v", err)
	}
	if !bytesEqual(w.b, data) {
		t.Fatalf("downloaded data mismatch after CDN fallback")
	}
}

// TestDownloadResumableTruncatesStalePart verifies that starting a fresh
// download truncates leftover part bytes: stale tail bytes beyond the
// expected size would otherwise fail the final size check forever.
func TestDownloadResumableTruncatesStalePart(t *testing.T) {
	data := make([]byte, 2*1024*1024+7)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	partPath := filepath.Join(dir, "test.bin.part")
	bitmapPath := ResumeStatePath(partPath)

	// Stale part file (e.g. from a previous, different download) with a
	// longer tail; no bitmap.
	if err := os.WriteFile(partPath, make([]byte, 5*1024*1024), 0o644); err != nil {
		t.Fatal(err)
	}
	partFile, err := os.OpenFile(partPath, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer partFile.Close()

	client := &serverLikeClient{data: data}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "test.bin")
	if err := DownloadResumable(context.Background(), file, partFile, 1, bitmapPath); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	stat, err := partFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() != int64(len(data)) {
		t.Fatalf("part size = %d, want %d (stale tail not truncated)", stat.Size(), len(data))
	}
	got := make([]byte, len(data))
	if _, err := partFile.ReadAt(got, 0); err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, data) {
		t.Fatalf("downloaded data mismatch")
	}
}

// oversizedChunkClient returns more bytes than the requested limit.
type oversizedChunkClient struct {
	*serverLikeClient
}

func (c *oversizedChunkClient) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	res, err := c.serverLikeClient.UploadGetFile(ctx, req)
	if f, ok := res.(*tg.UploadFile); ok && req.Offset == 0 {
		f.Bytes = append(f.Bytes, make([]byte, 16)...) // exceeds limit
	}
	return res, err
}

// TestDownloadResumableOversizedChunk rejects server responses that exceed
// the requested limit instead of letting them corrupt the file layout.
func TestDownloadResumableOversizedChunk(t *testing.T) {
	data := make([]byte, 1024*1024+7)
	for i := range data {
		data[i] = byte(i % 251)
	}
	dir := t.TempDir()
	bitmapPath := filepath.Join(dir, "test.bin.bitmap")
	w := &memWriterAt{b: make([]byte, len(data))}

	client := &oversizedChunkClient{serverLikeClient: &serverLikeClient{data: data}}
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, client, int64(len(data)), "test.bin")
	err := DownloadResumable(context.Background(), file, w, 1, bitmapPath)
	if err == nil {
		t.Fatalf("expected oversized chunk to fail")
	}
	if !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error %q does not mention the limit", err)
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
