package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

const testRecoverType = tasktype.TaskType("test-recover")

type stubTask struct {
	id string
}

func (s *stubTask) Type() tasktype.TaskType       { return testRecoverType }
func (s *stubTask) Title() string                 { return s.id }
func (s *stubTask) TaskID() string                { return s.id }
func (s *stubTask) Execute(context.Context) error { return nil }

type stubCodec struct{}

func (stubCodec) Marshal(task Executable) ([]byte, error) {
	return []byte(task.TaskID()), nil
}

func (stubCodec) Unmarshal(payload []byte) (Executable, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}
	return &stubTask{id: string(payload)}, nil
}

func initRecoveryEnv(t *testing.T) context.Context {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	content := fmt.Sprintf("[db]\npath = %q\n", filepath.Join(dir, "test.db"))
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.Init(context.Background(), cfgPath); err != nil {
		t.Fatalf("config init: %v", err)
	}
	database.Init(context.Background())
	RegisterTaskCodec(testRecoverType, stubCodec{})
	return context.Background()
}

func TestRecoverTasksReenqueuesAndMarksUnknownFailed(t *testing.T) {
	ctx := initRecoveryEnv(t)

	if err := database.CreateTask(ctx, &database.Task{
		ID: "rec-1", Type: string(testRecoverType), Payload: []byte("rec-1"), Status: string(database.TaskStatusQueued),
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateTask(ctx, &database.Task{
		ID: "rec-2", Type: string(testRecoverType), Payload: []byte("rec-2"), Status: string(database.TaskStatusRunning),
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateTask(ctx, &database.Task{
		ID: "drop-1", Type: "unregistered", Payload: nil, Status: string(database.TaskStatusQueued),
	}); err != nil {
		t.Fatal(err)
	}

	RecoverTasks(ctx)

	ids := map[string]bool{}
	for _, info := range GetQueuedTasks(ctx) {
		ids[info.ID] = true
	}
	if !ids["rec-1"] || !ids["rec-2"] {
		t.Fatalf("recovered task ids = %v, want rec-1 and rec-2", ids)
	}

	unfinished, err := database.GetUnfinishedTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(unfinished) != 2 {
		t.Fatalf("unfinished rows = %d, want 2", len(unfinished))
	}
	for _, task := range unfinished {
		if task.ID == "drop-1" {
			t.Fatalf("unregistered task record was not dropped")
		}
		if task.Status != string(database.TaskStatusQueued) {
			t.Fatalf("recovered task status = %s, want queued", task.Status)
		}
	}
	// The unrecoverable task must be kept and marked failed, not silently deleted.
	drop, err := database.GetTask(ctx, "drop-1")
	if err != nil {
		t.Fatalf("dropped task row missing: %v", err)
	}
	if drop.Status != string(database.TaskStatusFailed) {
		t.Fatalf("dropped task status = %s, want failed", drop.Status)
	}
	if drop.Error == "" {
		t.Fatalf("dropped task has no failure reason")
	}
}

func TestRecoverTasksMarksInvalidPayloadFailed(t *testing.T) {
	ctx := initRecoveryEnv(t)

	if err := database.CreateTask(ctx, &database.Task{
		ID: "bad-1", Type: string(testRecoverType), Payload: nil, Status: string(database.TaskStatusQueued),
	}); err != nil {
		t.Fatal(err)
	}
	RecoverTasks(ctx)

	// bad-1 must not be enqueued; its row is kept as failed.
	for _, info := range GetQueuedTasks(ctx) {
		if info.ID == "bad-1" {
			t.Fatalf("task with invalid payload was enqueued")
		}
	}
	count, err := database.CountUnfinishedTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unfinished rows = %d, want 0", count)
	}
	bad, err := database.GetTask(ctx, "bad-1")
	if err != nil {
		t.Fatalf("failed task row missing: %v", err)
	}
	if bad.Status != string(database.TaskStatusFailed) {
		t.Fatalf("bad task status = %s, want failed", bad.Status)
	}
}

func TestRecoverTasksSkipsAlreadyQueued(t *testing.T) {
	ctx := initRecoveryEnv(t)

	// A task submitted during startup is both persisted and in the queue.
	task := &stubTask{id: "live-1"}
	if err := AddTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	RecoverTasks(ctx)

	// The row must survive with its original status.
	row, err := database.GetTask(ctx, "live-1")
	if err != nil {
		t.Fatalf("row missing for queued task: %v", err)
	}
	if row.Status != string(database.TaskStatusQueued) {
		t.Fatalf("row status = %s, want queued", row.Status)
	}
}
