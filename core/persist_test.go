package core

import (
	"context"
	"encoding/json"
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

// doneStubTask reports its work finished in an earlier run; recovery must
// drop it instead of re-executing it (would duplicate the upload).
type doneStubTask struct {
	stubTask
}

func (d *doneStubTask) IsDone() bool { return true }

type doneStubCodec struct{}

func (doneStubCodec) Marshal(task Executable) ([]byte, error) {
	return []byte(task.TaskID()), nil
}

func (doneStubCodec) Unmarshal(payload []byte) (Executable, error) {
	return &doneStubTask{stubTask{id: string(payload)}}, nil
}

// TestDoneHelpersPreservePayload verifies AppendTaskDone and MarkTaskDone
// mutate only the "done" key and preserve the rest of the payload.
func TestDoneHelpersPreservePayload(t *testing.T) {
	ctx := initRecoveryEnv(t)

	if err := database.CreateTask(ctx, &database.Task{
		ID: "app-1", Type: "batch", Payload: []byte(`{"kind":"batch","id":"app-1","elements":[{"id":"e1"}]}`), Status: string(database.TaskStatusQueued),
	}); err != nil {
		t.Fatal(err)
	}
	if err := AppendTaskDone(ctx, "app-1", "e1"); err != nil {
		t.Fatal(err)
	}
	if err := AppendTaskDone(ctx, "app-1", "e1"); err != nil {
		t.Fatal(err)
	}
	if err := AppendTaskDone(ctx, "app-1", "e2"); err != nil {
		t.Fatal(err)
	}

	row, err := database.GetTask(ctx, "app-1")
	if err != nil {
		t.Fatal(err)
	}
	var batch struct {
		Kind     string `json:"kind"`
		Elements []struct {
			ID string `json:"id"`
		} `json:"elements"`
		Done []string `json:"done"`
	}
	if err := json.Unmarshal(row.Payload, &batch); err != nil {
		t.Fatal(err)
	}
	if batch.Kind != "batch" || len(batch.Elements) != 1 || batch.Elements[0].ID != "e1" {
		t.Fatalf("payload mutated beyond done: %s", row.Payload)
	}
	if len(batch.Done) != 2 || batch.Done[0] != "e1" || batch.Done[1] != "e2" {
		t.Fatalf("Done = %v, want [e1 e2] (no duplicates)", batch.Done)
	}

	if err := database.CreateTask(ctx, &database.Task{
		ID: "flag-1", Type: "file", Payload: []byte(`{"kind":"file","id":"flag-1","caption":"c"}`), Status: string(database.TaskStatusRunning),
	}); err != nil {
		t.Fatal(err)
	}
	if err := MarkTaskDone(ctx, "flag-1"); err != nil {
		t.Fatal(err)
	}
	row, err = database.GetTask(ctx, "flag-1")
	if err != nil {
		t.Fatal(err)
	}
	var single struct {
		Kind    string `json:"kind"`
		Caption string `json:"caption"`
		Done    bool   `json:"done"`
	}
	if err := json.Unmarshal(row.Payload, &single); err != nil {
		t.Fatal(err)
	}
	if !single.Done || single.Kind != "file" || single.Caption != "c" {
		t.Fatalf("MarkTaskDone payload = %s, want done=true with fields preserved", row.Payload)
	}
}
