package tdler

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"

	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

// serverLikeClient mimics real Telegram upload.getFile behavior: it returns
// up to limit bytes per chunk, and answers any offset at or past the end of
// the file with 400 OFFSET_INVALID.
type serverLikeClient struct {
	data []byte

	mu        sync.Mutex
	maxOffset int64
}

func (c *serverLikeClient) UploadGetFile(_ context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	c.mu.Lock()
	if req.Offset > c.maxOffset {
		c.maxOffset = req.Offset
	}
	c.mu.Unlock()
	if req.Offset >= int64(len(c.data)) {
		return nil, tgerr.New(400, "OFFSET_INVALID")
	}
	end := min(len(c.data), int(req.Offset)+req.Limit)
	return &tg.UploadFile{Bytes: c.data[req.Offset:end]}, nil
}

func (c *serverLikeClient) UploadGetFileHashes(context.Context, *tg.UploadGetFileHashesRequest) ([]tg.FileHash, error) {
	return nil, nil
}

func (c *serverLikeClient) UploadReuploadCDNFile(context.Context, *tg.UploadReuploadCDNFileRequest) ([]tg.FileHash, error) {
	return nil, nil
}

func (c *serverLikeClient) UploadGetCDNFileHashes(context.Context, *tg.UploadGetCDNFileHashesRequest) ([]tg.FileHash, error) {
	return nil, nil
}

func (c *serverLikeClient) UploadGetWebFile(context.Context, *tg.UploadGetWebFileRequest) (*tg.UploadWebFile, error) {
	return nil, nil
}

type memWriterAt struct {
	b []byte
}

func (w *memWriterAt) WriteAt(p []byte, off int64) (int, error) {
	copy(w.b[off:], p)
	return len(p), nil
}

func TestDownloadServerLikeEOF(t *testing.T) {
	const partSize = 1024 * 1024
	tests := []struct {
		name     string
		size     int
		parallel bool
	}{
		{"stream exact multiple of part size", 2 * partSize, false},
		{"stream non-multiple", 2*partSize + 12345, false},
		{"stream smaller than part size", 1234, false},
		{"parallel exact multiple of part size", 2 * partSize, true},
		{"parallel non-multiple", 2*partSize + 12345, true},
		{"parallel smaller than part size", 1234, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			for i := range data {
				data[i] = byte(i % 251)
			}
			client := &serverLikeClient{data: data}
			file := tfile.NewTGFile(
				&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2},
				client, int64(tt.size), "test.bin",
			)

			dl := NewDownloader(file)
			var got []byte
			var err error
			if tt.parallel {
				buf := make([]byte, tt.size)
				_, err = dl.WithThreads(4).Parallel(context.Background(), &memWriterAt{b: buf})
				got = buf
			} else {
				var buf bytes.Buffer
				_, err = dl.Stream(context.Background(), &buf)
				got = buf.Bytes()
			}
			if err != nil {
				t.Fatalf("download failed: %v", err)
			}
			if !bytes.Equal(got, data) {
				t.Fatalf("downloaded %d bytes, want %d matching bytes", len(got), len(data))
			}
			if client.maxOffset >= int64(tt.size) {
				t.Fatalf("requested offset %d at or past EOF (size %d)", client.maxOffset, tt.size)
			}
		})
	}
}
