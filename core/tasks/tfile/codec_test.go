package tfile

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/gotd/td/tg"

	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

type mockStorage struct{}

func (mockStorage) Init(context.Context, storconfig.StorageConfig) error { return nil }
func (mockStorage) Type() storenum.StorageType                           { return storenum.Local }
func (mockStorage) Name() string                                         { return "mock" }
func (mockStorage) Save(context.Context, io.Reader, string) error        { return nil }
func (mockStorage) Exists(context.Context, string) bool                  { return false }

var _ storage.Storage = mockStorage{}

// TestTaskPayloadRoundTrip verifies a recovered single-file task re-persists
// its done/overwrite/caption state: the rebuilt task has no original message
// to derive the caption from, and a bare ctx carries no overwrite flag.
func TestTaskPayloadRoundTrip(t *testing.T) {
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, nil, 100, "a.bin")
	task := &Task{
		ID:        "f-1",
		Ctx:       context.Background(),
		File:      file,
		Storage:   mockStorage{},
		Path:      "/p",
		overwrite: true,
		caption:   "cap",
		done:      true,
	}
	data, err := TaskCodec.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var p taskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !p.Done {
		t.Fatalf("Done = false, want true")
	}
	if !p.Overwrite {
		t.Fatalf("Overwrite = false, want true")
	}
	if p.Caption != "cap" {
		t.Fatalf("Caption = %q, want %q", p.Caption, "cap")
	}
}

// TestTaskCodecUnmarshalRequiresDownloader verifies recovery fails cleanly
// when no download client is registered (no bot connection).
func TestTaskCodecUnmarshalRequiresDownloader(t *testing.T) {
	data := []byte(`{"kind":"file","id":"f-1","file":{"kind":"document","id":1}}`)
	if _, err := TaskCodec.Unmarshal(data); err == nil {
		t.Fatalf("expected error without downloader client")
	}
}
