package localupload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func TestExecuteUploadsAndRemovesStaging(t *testing.T) {
	stageTask(t, "upload-task", []byte("payload"))

	target := &memoryStorage{name: "target"}
	var events []taskevent.Event
	ctx := taskevent.WithSink(context.Background(), taskevent.SinkFunc(func(event taskevent.Event) {
		events = append(events, event)
	}))
	task := NewTask("upload-task", "movie.mkv", 7, target, "movies", false)
	if err := task.Execute(ctx); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if target.path != path.Join("movies", "movie.mkv") {
		t.Fatalf("target path = %q", target.path)
	}
	if !bytes.Equal(target.data, []byte("payload")) {
		t.Fatalf("target data = %q", target.data)
	}
	payload, err := upload.StagingPayloadPath("upload-task")
	if err != nil {
		t.Fatalf("StagingPayloadPath: %v", err)
	}
	if _, err := os.Stat(payload); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged payload still exists: %v", err)
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
	task := NewTask("missing-task", "missing.txt", 1, &memoryStorage{name: "target"}, "", false)
	if err := task.Execute(context.Background()); err == nil {
		t.Fatal("Execute succeeded without staged payload")
	}
	results := task.ItemResults()
	if len(results) != 1 || results[0].Status != core.ItemStatusFailed {
		t.Fatalf("item results = %+v", results)
	}
}

func TestExecuteRejectsSizeMismatch(t *testing.T) {
	stageTask(t, "size-task", []byte("payload"))
	task := NewTask("size-task", "file.bin", 99, &memoryStorage{name: "target"}, "", false)
	if err := task.Execute(context.Background()); err == nil {
		t.Fatal("Execute accepted a size mismatch")
	}
	payload, err := upload.StagingPayloadPath("size-task")
	if err != nil {
		t.Fatalf("StagingPayloadPath: %v", err)
	}
	if _, err := os.Stat(payload); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed staged payload still exists: %v", err)
	}
}

func TestTaskParams(t *testing.T) {
	task := NewTask("params-task", "file.txt", 42, &memoryStorage{name: "target"}, "archive", true)
	raw, err := task.TaskParams()
	if err != nil {
		t.Fatalf("TaskParams: %v", err)
	}
	var spec taskspec.Upload
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("decode task params: %v", err)
	}
	if spec.Name != "file.txt" || spec.Size != 42 || spec.Storage != "target" || spec.Path != "archive" || !spec.Overwrite {
		t.Fatalf("task params = %+v", spec)
	}
}

func stageTask(t *testing.T, taskID string, data []byte) {
	t.Helper()
	setUploadTestConfig(t)
	payload, err := upload.StagingPayloadPath(taskID)
	if err != nil {
		t.Fatalf("StagingPayloadPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(payload), 0o700); err != nil {
		t.Fatalf("create staging dir: %v", err)
	}
	if err := os.WriteFile(payload, data, 0o600); err != nil {
		t.Fatalf("write staging payload: %v", err)
	}
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
