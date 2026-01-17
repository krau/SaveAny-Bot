package aria2dl

import (
	"context"
	"io"
	"testing"
	"time"

	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/aria2"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

type mockStorage struct {
	name     string
	savePath string
}

func (m *mockStorage) Name() string {
	return m.name
}

func (m *mockStorage) Type() storenum.StorageType {
	return storenum.StorageType("mock")
}

func (m *mockStorage) Init(ctx context.Context, config storconfig.StorageConfig) error {
	return nil
}

func (m *mockStorage) Save(ctx context.Context, reader io.Reader, path string) error {
	m.savePath = path
	return nil
}

func (m *mockStorage) Exists(ctx context.Context, path string) bool {
	return false
}

func (m *mockStorage) JoinStoragePath(path string) string {
	return path
}

type mockProgress struct {
	started  bool
	done     bool
	doneErr  error
	progress int
}

func (m *mockProgress) OnStart(ctx context.Context, task *Task) {
	m.started = true
}

func (m *mockProgress) OnProgress(ctx context.Context, task *Task, status *aria2.Status) {
	m.progress++
}

func (m *mockProgress) OnDone(ctx context.Context, task *Task, err error) {
	m.done = true
	m.doneErr = err
}

func TestTaskCreation(t *testing.T) {
	ctx := context.Background()
	mockStor := &mockStorage{name: "test-storage"}
	mockProg := &mockProgress{}

	task := NewTask(
		"test-task-id",
		ctx,
		"test-gid",
		[]string{"http://example.com/file.zip"},
		nil,
		mockStor,
		"/test/path",
		mockProg,
	)

	if task.ID != "test-task-id" {
		t.Errorf("Expected task ID to be 'test-task-id', got '%s'", task.ID)
	}

	if task.GID != "test-gid" {
		t.Errorf("Expected GID to be 'test-gid', got '%s'", task.GID)
	}

	if task.Type() != tasktype.TaskTypeAria2 {
		t.Errorf("Expected task type to be TaskTypeAria2, got '%s'", task.Type())
	}

	if task.TaskID() != "test-task-id" {
		t.Errorf("Expected TaskID() to return 'test-task-id', got '%s'", task.TaskID())
	}

	if task.Storage.Name() != "test-storage" {
		t.Errorf("Expected storage name to be 'test-storage', got '%s'", task.Storage.Name())
	}
}

func TestProgressTracker(t *testing.T) {
	ctx := context.Background()
	mockStor := &mockStorage{name: "test-storage"}
	mockProg := &mockProgress{}

	task := NewTask(
		"test-task-id",
		ctx,
		"test-gid",
		[]string{"http://example.com/file.zip"},
		nil,
		mockStor,
		"/test/path",
		mockProg,
	)

	// Test OnStart
	mockProg.OnStart(ctx, task)
	if !mockProg.started {
		t.Error("Expected OnStart to set started to true")
	}

	// Test OnProgress
	status := &aria2.Status{
		GID:             "test-gid",
		Status:          "active",
		TotalLength:     "1000000",
		CompletedLength: "500000",
		DownloadSpeed:   "100000",
	}
	mockProg.OnProgress(ctx, task, status)
	if mockProg.progress != 1 {
		t.Errorf("Expected progress to be 1, got %d", mockProg.progress)
	}

	// Test OnDone
	mockProg.OnDone(ctx, task, nil)
	if !mockProg.done {
		t.Error("Expected OnDone to set done to true")
	}
	if mockProg.doneErr != nil {
		t.Errorf("Expected doneErr to be nil, got %v", mockProg.doneErr)
	}
}

func TestTaskTitle(t *testing.T) {
	ctx := context.Background()
	mockStor := &mockStorage{name: "test-storage"}

	task := NewTask(
		"test-task-id",
		ctx,
		"test-gid-123",
		[]string{"http://example.com/file.zip"},
		nil,
		mockStor,
		"/test/path",
		nil,
	)

	title := task.Title()
	expectedSubstr := "test-gid-123"
	if len(title) == 0 {
		t.Error("Expected title to not be empty")
	}

	// Check if title contains the GID
	found := false
	for i := 0; i < len(title)-len(expectedSubstr)+1; i++ {
		if title[i:i+len(expectedSubstr)] == expectedSubstr {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected title to contain GID '%s', got '%s'", expectedSubstr, title)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockStor := &mockStorage{name: "test-storage"}
	mockProg := &mockProgress{}

	task := NewTask(
		"test-task-id",
		ctx,
		"test-gid",
		[]string{"http://example.com/file.zip"},
		nil, // nil client will cause Execute to fail/timeout
		mockStor,
		"/test/path",
		mockProg,
	)

	// Just verify the task structure is valid
	if task.ctx.Err() != nil {
		t.Error("Context should not be cancelled yet")
	}

	// Wait for context to timeout
	<-ctx.Done()
	if ctx.Err() == nil {
		t.Error("Context should be cancelled after timeout")
	}
}
