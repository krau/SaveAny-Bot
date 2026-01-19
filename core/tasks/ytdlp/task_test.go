package ytdlp

import (
	"context"
	"io"
	"testing"

	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

// MockStorage is a simple mock for testing
type MockStorage struct{}

func (m *MockStorage) Init(ctx context.Context, cfg storcfg.StorageConfig) error     { return nil }
func (m *MockStorage) Type() storenum.StorageType                                    { return "mock" }
func (m *MockStorage) Name() string                                                  { return "test-storage" }
func (m *MockStorage) JoinStoragePath(p string) string                               { return "test-path" }
func (m *MockStorage) Save(ctx context.Context, reader io.Reader, path string) error { return nil }
func (m *MockStorage) Exists(ctx context.Context, path string) bool                  { return false }

func TestNewTask(t *testing.T) {
	ctx := context.Background()
	urls := []string{"https://example.com/video"}
	flags := []string{"--format", "best"}
	stor := &MockStorage{}
	storPath := "test-path"

	task := NewTask("test-id", ctx, urls, flags, stor, storPath, nil)

	if task == nil {
		t.Fatal("NewTask returned nil")
	}

	if task.ID != "test-id" {
		t.Errorf("Expected task ID 'test-id', got '%s'", task.ID)
	}

	if len(task.URLs) != 1 || task.URLs[0] != "https://example.com/video" {
		t.Errorf("Expected URLs to contain 'https://example.com/video', got %v", task.URLs)
	}

	if len(task.Flags) != 2 || task.Flags[0] != "--format" || task.Flags[1] != "best" {
		t.Errorf("Expected flags to contain '--format' and 'best', got %v", task.Flags)
	}

	if task.Storage.Name() != "test-storage" {
		t.Errorf("Expected storage name 'test-storage', got '%s'", task.Storage.Name())
	}
}

func TestNewTaskWithoutFlags(t *testing.T) {
	ctx := context.Background()
	urls := []string{"https://example.com/video1", "https://example.com/video2"}
	var flags []string // No flags
	stor := &MockStorage{}
	storPath := "test-path"

	task := NewTask("test-id-2", ctx, urls, flags, stor, storPath, nil)

	if task == nil {
		t.Fatal("NewTask returned nil")
	}

	if len(task.URLs) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(task.URLs))
	}

	if len(task.Flags) != 0 {
		t.Errorf("Expected 0 flags, got %d", len(task.Flags))
	}
}

func TestTaskTitle(t *testing.T) {
	ctx := context.Background()
	stor := &MockStorage{}

	// Test with single URL
	task1 := NewTask("id1", ctx, []string{"https://example.com/video"}, nil, stor, "path", nil)
	title1 := task1.Title()
	if title1 == "" {
		t.Error("Task title should not be empty")
	}

	// Test with multiple URLs
	task2 := NewTask("id2", ctx, []string{"https://example.com/v1", "https://example.com/v2"}, nil, stor, "path", nil)
	title2 := task2.Title()
	if title2 == "" {
		t.Error("Task title should not be empty")
	}
}

func TestTaskType(t *testing.T) {
	ctx := context.Background()
	stor := &MockStorage{}
	task := NewTask("id", ctx, []string{"https://example.com"}, nil, stor, "path", nil)

	taskType := task.Type()
	if taskType.String() != "ytdlp" {
		t.Errorf("Expected task type 'ytdlp', got '%s'", taskType.String())
	}
}

func TestTaskID(t *testing.T) {
	ctx := context.Background()
	stor := &MockStorage{}
	expectedID := "test-task-id-123"

	task := NewTask(expectedID, ctx, []string{"https://example.com"}, nil, stor, "path", nil)

	if task.TaskID() != expectedID {
		t.Errorf("Expected task ID '%s', got '%s'", expectedID, task.TaskID())
	}
}
