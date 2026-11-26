package localupload

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/krau/SaveAny-Bot/common/upload"
	"github.com/krau/SaveAny-Bot/config"
	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/core"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
)

func TestExecuteUsesOriginalStagingRoot(t *testing.T) {
	root, payload := stageTask(t, "upload-task", []byte("payload"))
	setUploadTestConfig(t)

	target := &memoryStorage{name: "target"}
	var events []taskevent.Event
	ctx := taskevent.WithSink(context.Background(), taskevent.SinkFunc(func(event taskevent.Event) {
		events = append(events, event)
	}))
	task := NewTask("upload-task", root, "movie.mkv", 7, target, "movies", false)
	if err := task.Execute(ctx); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if target.path != path.Join("movies", "movie.mkv") {
		t.Fatalf("target path = %q", target.path)
	}
	if !bytes.Equal(target.data, []byte("payload")) {
		t.Fatalf("target data = %q", target.data)
	}
	if _, err := os.Stat(payload); err != nil {
		t.Fatalf("staged payload removed before completion was persisted: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no progress events emitted")
	}
	results := task.ItemResults()
	if len(results) != 1 || results[0].Status != core.ItemStatusCompleted {
		t.Fatalf("item results = %+v", results)
	}
}

func TestExecuteFailsWhenStagingIsMissing(t *testing.T) {
	setUploadTestConfig(t)
	task := NewTask("missing-task", "", "missing.txt", 1, &memoryStorage{name: "target"}, "", false)
	if err := task.Execute(context.Background()); err == nil {
		t.Fatal("Execute succeeded without staged payload")
	}
	results := task.ItemResults()
	if len(results) != 1 || results[0].Status != core.ItemStatusFailed {
		t.Fatalf("item results = %+v", results)
	}
}

func TestExecuteRejectsSizeMismatch(t *testing.T) {
	root, payload := stageTask(t, "size-task", []byte("payload"))
	task := NewTask("size-task", root, "file.bin", 99, &memoryStorage{name: "target"}, "", false)
	if err := task.Execute(context.Background()); err == nil {
		t.Fatal("Execute accepted a size mismatch")
	}
	if _, err := os.Stat(payload); err != nil {
		t.Fatalf("staged payload removed before failure was persisted: %v", err)
	}
}

func TestTaskParams(t *testing.T) {
	task := NewTask("params-task", "staging-root", "file.txt", 42, &memoryStorage{name: "target"}, "archive", true)
	raw, err := task.TaskParams()
	if err != nil {
		t.Fatalf("TaskParams: %v", err)
	}
	var spec taskspec.Upload
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("decode task params: %v", err)
	}
	if spec.Name != "file.txt" || spec.Size != 42 || spec.Storage != "target" || spec.Path != "archive" || !spec.Overwrite || spec.StagingRoot != "staging-root" {
		t.Fatalf("task params = %+v", spec)
	}
}

func stageTask(t *testing.T, taskID string, data []byte) (string, string) {
	t.Helper()
	setUploadTestConfig(t)
	root, err := upload.StagingRoot()
	if err != nil {
		t.Fatalf("StagingRoot: %v", err)
	}
	payload, err := upload.StagingPayloadPath(root, taskID)
	if err != nil {
		t.Fatalf("StagingPayloadPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(payload), 0o700); err != nil {
		t.Fatalf("create staging dir: %v", err)
	}
	if err := os.WriteFile(payload, data, 0o600); err != nil {
		t.Fatalf("write staging payload: %v", err)
	}
	return root, payload
}

func setUploadTestConfig(t *testing.T) {
	t.Helper()
	previous := config.C()
	cfg := previous
	cfg.Temp.BasePath = t.TempDir()
	cfg.Retry = 2
	config.Set(cfg)
	t.Cleanup(func() { config.Set(previous) })
}

type memoryStorage struct {
	name string
	path string
	data []byte
}

func (s *memoryStorage) Init(context.Context, storcfg.StorageConfig) error { return nil }
func (s *memoryStorage) Type() storenum.StorageType                        { return storenum.Local }
func (s *memoryStorage) Name() string                                      { return s.name }
func (s *memoryStorage) Exists(context.Context, string) (bool, error)      { return false, nil }
func (s *memoryStorage) Save(_ context.Context, reader io.Reader, storagePath string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.path = storagePath
	s.data = data
	return nil
}
