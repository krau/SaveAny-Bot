package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) {
	t.Helper()
	d, err := gorm.Open(gormlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := d.AutoMigrate(&Task{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	old := db
	db = d
	t.Cleanup(func() { db = old })
}

func TestTaskCRUD(t *testing.T) {
	newTestDB(t)
	ctx := context.Background()

	task := &Task{
		ID:      "task-1",
		Type:    "tfile",
		Payload: []byte(`{"file":"x"}`),
		Status:  string(TaskStatusQueued),
	}
	if err := CreateTask(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}

	unfinished, err := GetUnfinishedTasks(ctx)
	if err != nil {
		t.Fatalf("get unfinished: %v", err)
	}
	if len(unfinished) != 1 || unfinished[0].ID != "task-1" {
		t.Fatalf("got %+v, want 1 task task-1", unfinished)
	}

	if err := UpdateTaskStatus(ctx, "task-1", TaskStatusRunning, ""); err != nil {
		t.Fatalf("update: %v", err)
	}
	unfinished, err = GetUnfinishedTasks(ctx)
	if err != nil {
		t.Fatalf("get unfinished after update: %v", err)
	}
	if len(unfinished) != 1 || unfinished[0].Status != string(TaskStatusRunning) {
		t.Fatalf("running status not persisted: %+v", unfinished)
	}

	if err := DeleteTask(ctx, "task-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	count, err := CountUnfinishedTasks(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestTaskUpsert(t *testing.T) {
	newTestDB(t)
	ctx := context.Background()

	task := &Task{ID: "task-2", Type: "tfile", Status: string(TaskStatusQueued)}
	if err := UpsertTask(ctx, task); err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	task.Status = string(TaskStatusRunning)
	task.Payload = []byte("new")
	if err := UpsertTask(ctx, task); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	unfinished, err := GetUnfinishedTasks(ctx)
	if err != nil {
		t.Fatalf("get unfinished: %v", err)
	}
	if len(unfinished) != 1 || unfinished[0].Status != string(TaskStatusRunning) || string(unfinished[0].Payload) != "new" {
		t.Fatalf("upsert did not replace: %+v", unfinished)
	}
}

func TestGetUnfinishedTasksExcludesFinished(t *testing.T) {
	newTestDB(t)
	ctx := context.Background()

	if err := CreateTask(ctx, &Task{ID: "done", Type: "tfile", Status: string(TaskStatusFailed)}); err != nil {
		t.Fatal(err)
	}
	if err := CreateTask(ctx, &Task{ID: "pending", Type: "tfile", Status: string(TaskStatusQueued)}); err != nil {
		t.Fatal(err)
	}
	unfinished, err := GetUnfinishedTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(unfinished) != 1 || unfinished[0].ID != "pending" {
		t.Fatalf("got %+v, want only pending", unfinished)
	}
}
