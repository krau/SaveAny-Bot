package queue_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/queue"
)

// helper to create a simple Task with integer payload
func newTask(id string) *queue.Task[int] {
	return queue.NewTask(context.Background(), id, "testing", 0)
}

func TestAddAndLength(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	if q.Length() != 0 {
		t.Fatalf("expected length 0, got %d", q.Length())
	}
	t1 := newTask("t1")
	if err := q.Add(t1); err != nil {
		t.Fatalf("unexpected error on Add: %v", err)
	}
	if q.Length() != 1 {
		t.Fatalf("expected length 1, got %d", q.Length())
	}
}

func TestDuplicateAdd(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	t1 := newTask("dup")
	if err := q.Add(t1); err != nil {
		t.Fatalf("unexpected error on first Add: %v", err)
	}
	if err := q.Add(t1); err == nil {
		t.Fatal("expected error on duplicate Add, got nil")
	}
}

func TestCancelAndActiveLength(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	t1 := newTask("1")
	t2 := newTask("2")
	q.Add(t1)
	q.Add(t2)
	// Cancel t1
	if err := q.CancelTask("1"); err != nil {
		t.Fatalf("unexpected error on CancelTask: %v", err)
	}
	// Length counts all entries
	if q.Length() != 2 {
		t.Fatalf("expected total length 2, got %d", q.Length())
	}
	// ActiveLength skips cancelled
	if got := q.ActiveLength(); got != 1 {
		t.Fatalf("expected active length 1, got %d", got)
	}
}

func TestCloseBehavior(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	done := make(chan struct{})
	// consumer
	go func() {
		_, err := q.Get()
		if !errors.Is(err, queue.ErrQueueClosed) {
			t.Errorf("expected ErrQueueClosed from closed empty queue, got %v", err)
		}
		close(done)
	}()
	// allow goroutine to block

	// close queue
	q.Close()
	<-done
}

// Regression: Get() must not deadlock when every queued task was cancelled
// before a worker picked it up (previously recursed while holding the mutex).
func TestGetAfterAllCancelled(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	for i := range 3 {
		if err := q.Add(newTask(fmt.Sprintf("c%d", i))); err != nil {
			t.Fatalf("unexpected error on Add: %v", err)
		}
	}
	for i := range 3 {
		if err := q.CancelTask(fmt.Sprintf("c%d", i)); err != nil {
			t.Fatalf("unexpected error on CancelTask: %v", err)
		}
	}

	// A task added after the cancelled ones must still be delivered.
	if err := q.Add(newTask("late")); err != nil {
		t.Fatalf("unexpected error on Add after cancel: %v", err)
	}

	done := make(chan struct{})
	var got *queue.Task[int]
	go func() {
		var err error
		got, err = q.Get()
		if err != nil {
			t.Errorf("unexpected error on Get: %v", err)
		}
		close(done)
	}()
	select {
	case <-done:
		if got == nil || got.ID != "late" {
			t.Fatalf("expected task 'late', got %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Get() deadlocked after all queued tasks were cancelled")
	}
}

// Cancelled queued tasks must be dropped from the task map so their TaskID
// can be reused (previously they leaked until process exit).
func TestCancelledQueuedTaskIDReusable(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	if err := q.Add(newTask("dup")); err != nil {
		t.Fatal(err)
	}
	if err := q.CancelTask("dup"); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var gotID string
	go func() {
		task, err := q.Get()
		if err != nil {
			t.Errorf("unexpected error on Get: %v", err)
			close(done)
			return
		}
		gotID = task.ID
		close(done)
	}()
	// The first Get skips the cancelled task and blocks; a live task unblocks it.
	if err := q.Add(newTask("late")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
		if gotID != "late" {
			t.Fatalf("expected 'late', got %q", gotID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Get did not return the live task")
	}
	// The cancelled task was dropped from the map: its ID is reusable.
	if err := q.Add(newTask("dup")); err != nil {
		t.Fatalf("expected cancelled task ID to be reusable, got: %v", err)
	}
}

func TestConcurrencySafety(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	var wg sync.WaitGroup
	n := 1000
	// producers
	wg.Go(func() {
		for i := range n {
			q.Add(newTask(fmt.Sprintf("p%d", i)))
		}
	})
	// consumers
	wg.Go(func() {
		count := 0
		for count < n {
			_, err := q.Get()
			if err != nil {
				continue
			}
			count++
		}
	})
	wg.Wait()
}
