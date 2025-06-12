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

	if task.IsCancelled() {
		return fmt.Errorf("task %s has been cancelled", task.ID)
	}

	element := tq.tasks.PushBack(task)
	task.element = element
	tq.taskMap[task.ID] = task

	tq.cond.Signal()
	return nil
}

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

		if !task.IsCancelled() {
			tq.runningTaskMap[task.ID] = task
			return task, nil
		}
	}

	if !tq.closed {
		return tq.Get()
	}

	return nil, fmt.Errorf("queue is closed and empty")
}

func (tq *TaskQueue[T]) Done(taskID string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	delete(tq.taskMap, taskID)
	delete(tq.runningTaskMap, taskID)
}

func (tq *TaskQueue[T]) Peek() (*Task[T], error) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	if tq.tasks.Len() == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	for element := tq.tasks.Front(); element != nil; element = element.Next() {
		task := element.Value.(*Task[T])
		if !task.IsCancelled() {
			return task, nil
		}
	}

	return nil, fmt.Errorf("queue has no valid tasks")
}

func (tq *TaskQueue[T]) Length() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	return tq.tasks.Len()
}

func (tq *TaskQueue[T]) ActiveLength() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	count := 0
	for element := tq.tasks.Front(); element != nil; element = element.Next() {
		task := element.Value.(*Task[T])
		if !task.IsCancelled() {
			count++
		}
	}
	return count
}

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

func (tq *TaskQueue[T]) RemoveTask(taskID string) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	task, exists := tq.taskMap[taskID]
	if !exists {
		_, exists = tq.runningTaskMap[taskID]
		if exists {
			delete(tq.runningTaskMap, taskID)
		}
		return fmt.Errorf("task %s is already running, cannot remove from queue", taskID)
	}

	if task.element != nil {
		tq.tasks.Remove(task.element)
	}
	delete(tq.taskMap, taskID)
	task.Cancel()
	return nil
}

func (tq *TaskQueue[T]) CancelAll() {
	tq.mu.RLock()
	tasks := make([]*Task[T], 0, tq.tasks.Len())
	for element := tq.tasks.Front(); element != nil; element = element.Next() {
		tasks = append(tasks, element.Value.(*Task[T]))
	}
	tq.mu.RUnlock()

	for _, task := range tasks {
		task.Cancel()
	}
}

func (tq *TaskQueue[T]) GetTask(taskID string) (*Task[T], error) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	task, exists := tq.taskMap[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s does not exist", taskID)
	}

	return task, nil
}

func (tq *TaskQueue[T]) Close() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.closed = true
	tq.cond.Broadcast()
}

func (tq *TaskQueue[T]) IsClosed() bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	return tq.closed
}

func (tq *TaskQueue[T]) Clear() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for element := tq.tasks.Front(); element != nil; element = element.Next() {
		task := element.Value.(*Task[T])
		task.Cancel()
	}

	tq.tasks.Init()
	tq.taskMap = make(map[string]*Task[T])
}

func (tq *TaskQueue[T]) CleanupCancelled() int {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	removed := 0
	element := tq.tasks.Front()

	for element != nil {
		next := element.Next()
		task := element.Value.(*Task[T])

		if task.IsCancelled() {
			tq.tasks.Remove(element)
			delete(tq.taskMap, task.ID)
			removed++
		}

		element = next
	}

	return removed
}
