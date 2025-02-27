package queue

import (
	"container/list"
	"sync"

	"github.com/krau/SaveAny-Bot/types"
)

type TaskQueue struct {
	list      *list.List
	cond      *sync.Cond
	mutex     *sync.Mutex
	activeMap map[string]*types.Task
}

func (q *TaskQueue) AddTask(task *types.Task) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if task.Status == types.Pending {
		q.list.PushBack(task)
		q.cond.Signal()
	} else {
		delete(q.activeMap, task.Key())
	}
}

func (q *TaskQueue) GetTask() *types.Task {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	for q.list.Len() == 0 {
		q.cond.Wait()
	}
	e := q.list.Front()
	task := e.Value.(*types.Task)
	q.list.Remove(e)
	q.activeMap[task.Key()] = task
	return task
}

func (q *TaskQueue) CancelTask(key string) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if task, ok := q.activeMap[key]; ok {
		if task.Cancel != nil {
			task.Cancel()
			return true
		}
	}
	for e := q.list.Front(); e != nil; e = e.Next() {
		task := e.Value.(*types.Task)
		if task.Key() == key {
			if task.Cancel != nil {
				task.Cancel()
			}
			q.list.Remove(e)
			return true
		}
	}
	return false
}

func (q *TaskQueue) Len() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.list.Len()
}

var Queue *TaskQueue

func init() {
	Queue = NewQueue()
}

func NewQueue() *TaskQueue {
	m := &sync.Mutex{}
	return &TaskQueue{
		list:      list.New(),
		cond:      sync.NewCond(m),
		mutex:     m,
		activeMap: make(map[string]*types.Task),
	}
}

func AddTask(task *types.Task) {
	Queue.AddTask(task)
}

func GetTask() *types.Task {
	return Queue.GetTask()
}

func Len() int {
	return Queue.Len()
}

func CancelTask(key string) bool {
	return Queue.CancelTask(key)
}
