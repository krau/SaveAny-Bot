package queue_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/krau/SaveAny-Bot/pkg/queue"
)

// helper to create a simple Task with integer payload
func newTask(id string) *queue.Task[int] {
	return queue.NewTask(context.Background(), id, 0)
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

func TestGetAndPeek(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	t1 := newTask("a")
	t2 := newTask("b")
	q.Add(t1)
	q.Add(t2)
	// Peek should return t1
	peeked, err := q.Peek()
	if err != nil {
		t.Fatalf("unexpected error on Peek: %v", err)
	}
	if peeked.ID != "a" {
		t.Fatalf("expected Peek ID 'a', got '%s'", peeked.ID)
	}
	// Get should return t1 then t2
	first, err := q.Get()
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if first.ID != "a" {
		t.Fatalf("expected first Get ID 'a', got '%s'", first.ID)
	}
	second, err := q.Get()
	if err != nil {
		t.Fatalf("unexpected error on second Get: %v", err)
	}
	if second.ID != "b" {
		t.Fatalf("expected second Get ID 'b', got '%s'", second.ID)
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

func TestRemoveTask(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	t1 := newTask("r1")
	q.Add(t1)
	if err := q.RemoveTask("r1"); err != nil {
		t.Fatalf("unexpected error on RemoveTask: %v", err)
	}
	if q.Length() != 0 {
		t.Fatalf("expected length 0 after remove, got %d", q.Length())
	}
}

func TestClearAndCleanupCancelled(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	tasks := []*queue.Task[int]{newTask("c1"), newTask("c2"), newTask("c3")}
	for _, tsk := range tasks {
		q.Add(tsk)
	}
	// Cancel one
	q.CancelTask("c2")
	// Cleanup cancelled
	removed := q.CleanupCancelled()
	if removed != 1 {
		t.Fatalf("expected removed 1, got %d", removed)
	}
	if q.ActiveLength() != 2 {
		t.Fatalf("expected active length 2 after cleanup, got %d", q.ActiveLength())
	}
	// Clear all
	q.Clear()
	if q.Length() != 0 {
		t.Fatalf("expected length 0 after clear, got %d", q.Length())
	}
}

func TestCloseBehavior(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	done := make(chan struct{})
	// consumer
	go func() {
		_, err := q.Get()
		if err == nil {
			t.Errorf("expected error when getting from closed empty queue, got nil")
		}
		close(done)
	}()
	// allow goroutine to block

	// close queue
	q.Close()
	<-done
}

func TestConcurrencySafety(t *testing.T) {
	q := queue.NewTaskQueue[int]()
	var wg sync.WaitGroup
	n := 1000
	// producers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			q.Add(newTask(fmt.Sprintf("p%d", i)))
		}
	}()
	// consumers
	wg.Add(1)
	go func() {
		defer wg.Done()
		count := 0
		for count < n {
			_, err := q.Get()
			if err != nil {
				continue
			}
			count++
		}
	}()
	wg.Wait()
}
