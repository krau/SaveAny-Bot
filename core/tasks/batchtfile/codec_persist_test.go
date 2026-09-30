package batchtfile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gotd/td/tg"

	"github.com/krau/SaveAny-Bot/config"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/database"
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

// TestBatchCodecMarshalKeepsDoneList verifies a re-persist merges the done
// list from the previous payload with newly completed elements. Without the
// merge, a second restart would re-upload elements that already finished.
func TestBatchCodecMarshalKeepsDoneList(t *testing.T) {
	file := tfile.NewTGFile(&tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, nil, 100, "a.bin")
	task := &Task{
		ID:       "batch-1",
		ctx:      context.Background(),
		elems:    []TaskElement{{ID: "e-live", Storage: mockStorage{}, Path: "/p", File: file}},
		doneList: []string{"e-done-prev"},
	}
	states, index := newItemProgressStates([]TaskElement{{ID: "e-live"}})
	task.itemStates = states
	task.itemIndex = index
	task.markItemCompleted("e-live")

	data, err := batchCodec{}.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var p taskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	got := make(map[string]bool, len(p.Done))
	for _, id := range p.Done {
		got[id] = true
	}
	if !got["e-done-prev"] || !got["e-live"] {
		t.Fatalf("Done = %v, want both e-done-prev and e-live", p.Done)
	}
}

// TestBatchCodecMarshalOverwrite verifies the recovered overwrite flag
// survives a re-persist even though the recovered task's ctx is bare.
func TestBatchCodecMarshalOverwrite(t *testing.T) {
	task := &Task{
		ID:        "batch-1",
		ctx:       context.Background(),
		overwrite: true,
	}
	data, err := batchCodec{}.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var p taskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !p.Overwrite {
		t.Fatalf("Overwrite = false, want true")
	}
}

// TestBatchCodecUnmarshalRequiresDownloader verifies recovery fails cleanly
// when no download client is registered (no bot connection).
func TestBatchCodecUnmarshalRequiresDownloader(t *testing.T) {
	data := []byte(`{"kind":"batch","id":"b-1","elements":[]}`)
	if _, err := (batchCodec{}).Unmarshal(data); err == nil {
		t.Fatalf("expected error without downloader client")
	}
}

func TestPersistElementDoneConcurrent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	// workers=1 keeps visibleActiveItems() at 1 for later tests, matching the
	// uninitialized-config behavior they were written against.
	content := fmt.Sprintf("workers = 1\n\n[db]\npath = %q\n", filepath.Join(dir, "test.db"))
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.Init(context.Background(), cfgPath); err != nil {
		t.Fatalf("config init: %v", err)
	}
	database.Init(context.Background())

	task := &Task{ID: "t-1"}
	payload, err := json.Marshal(taskPayload{Kind: "batch", ID: "t-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CreateTask(context.Background(), &database.Task{
		ID: "t-1", Type: "tgfiles", Payload: payload, Status: string(database.TaskStatusQueued),
	}); err != nil {
		t.Fatal(err)
	}

	const count = 20
	var wg sync.WaitGroup
	for i := range count {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			task.persistElementDone(context.Background(), fmt.Sprintf("e-%d", n))
		}(i)
	}
	wg.Wait()

	row, err := database.GetTask(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	var p taskPayload
	if err := json.Unmarshal(row.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(p.Done) != count {
		t.Fatalf("Done has %d entries, want %d (lost updates: %v)", len(p.Done), count, p.Done)
	}
}
