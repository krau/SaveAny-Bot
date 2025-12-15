package queue

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
)

type TaskQueue[T any] struct {
	tasks          *list.List
	taskMap        map[string]*Task[T]
	runningTaskMap map[string]*Task[T]
	mu             sync.RWMutex
	cond           *sync.Cond
	closed         bool
}

func NewTaskQueue[T any]() *TaskQueue[T] {
	tq := &TaskQueue[T]{
		tasks:          list.New(),
		taskMap:        make(map[string]*Task[T]),
		runningTaskMap: make(map[string]*Task[T]),
	}
	tq.cond = sync.NewCond(&tq.mu)
	return tq
}

func (tq *TaskQueue[T]) Add(task *Task[T]) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.closed {
		return errors.New("queue is closed")
	}

	if _, exists := tq.taskMap[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	if task.Cancelled() {
		return fmt.Errorf("task %s has been cancelled", task.ID)
	}

	element := tq.tasks.PushBack(task)
	task.element = element
	tq.taskMap[task.ID] = task

	tq.cond.Signal()
	return nil
}

// Get retrieves and removes the next non-cancelled task from the queue, adding it to the running tasks.
// Blocks until a task is available or the queue is closed.
func (tq *TaskQueue[T]) Get() (*Task[T], error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for tq.tasks.Len() == 0 && !tq.closed {
		tq.cond.Wait()
	}

	if tq.closed && tq.tasks.Len() == 0 {
		return nil, fmt.Errorf("queue is closed and empty")
	}

	for tq.tasks.Len() > 0 {
		element := tq.tasks.Front()
		task := element.Value.(*Task[T])

		tq.tasks.Remove(element)
		task.element = nil

		if !task.Cancelled() {
			tq.runningTaskMap[task.ID] = task
			return task, nil
		}
	}

	if !tq.closed {
		return tq.Get()
	}

	return nil, fmt.Errorf("queue is closed and empty")
}

// Done stops(cancels) and removes the task from the running tasks.
func (tq *TaskQueue[T]) Done(taskID string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	delete(tq.taskMap, taskID)
	delete(tq.runningTaskMap, taskID)
}

func (tq *TaskQueue[T]) Length() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	return tq.tasks.Len()
}

// ActiveLength returns the number of non-cancelled tasks in the queue.
func (tq *TaskQueue[T]) ActiveLength() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	count := 0
	for element := tq.tasks.Front(); element != nil; element = element.Next() {
		task := element.Value.(*Task[T])
		if !task.Cancelled() {
			count++
		}
	}
	return count
}

// RunningTasks returns the currently running tasks' info.
func (tq *TaskQueue[T]) RunningTasks() []TaskInfo {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	tasks := make([]TaskInfo, 0, len(tq.runningTaskMap))
	for _, task := range tq.runningTaskMap {
		if task.Cancelled() {
			continue
		}
		tasks = append(tasks, TaskInfo{
			ID:        task.ID,
			Title:     task.Title,
			Created:   task.created,
			Cancelled: task.Cancelled(),
		})
	}
	return tasks
}

// QueuedTasks returns the queued (not yet running) tasks' info.
// The sorting is in the order of addition.
func (tq *TaskQueue[T]) QueuedTasks() []TaskInfo {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	tasks := make([]TaskInfo, 0, tq.tasks.Len())
	for element := tq.tasks.Front(); element != nil; element = element.Next() {
		task := element.Value.(*Task[T])
		if !task.Cancelled() {
			tasks = append(tasks, TaskInfo{
				ID:        task.ID,
				Title:     task.Title,
				Created:   task.created,
				Cancelled: task.Cancelled(),
			})
		}
	}
	return tasks
}

// CancelTask cancels a task by its ID.
// It looks for the task in both queued and running tasks.
// [NOTE] Cancelled tasks will not be removed from the queue, but marked as cancelled. Use Done to remove them.
// [WARN] Cancelling a running task relies on the task's implementation to respect the cancellation. If the task does not check for cancellation, it may continue running.
func (tq *TaskQueue[T]) CancelTask(taskID string) error {
	tq.mu.RLock()
	task, exists := tq.taskMap[taskID]
	if !exists {
		task, exists = tq.runningTaskMap[taskID]
	}
	tq.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s does not exist", taskID)
	}

	task.Cancel()
	return nil
}

func (tq *TaskQueue[T]) Close() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.closed = true
	tq.cond.Broadcast()
}
