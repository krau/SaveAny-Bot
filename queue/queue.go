package queue

import (
	"container/list"
	"sync"

	"github.com/krau/SaveAny-Bot/types"
)

type TaskQueue struct {
	list  *list.List
	cond  *sync.Cond
	mutex *sync.Mutex
}

func (q *TaskQueue) AddTask(task types.Task) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.list.PushBack(task)
	q.cond.Signal()
}

func (q *TaskQueue) GetTask() types.Task {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	for q.list.Len() == 0 {
		q.cond.Wait()
	}
	e := q.list.Front()
	task := e.Value.(types.Task)
	q.list.Remove(e)
	return task
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
		list:  list.New(),
		cond:  sync.NewCond(m),
		mutex: m,
	}
}

func AddTask(task types.Task) {
	Queue.AddTask(task)
}

func GetTask() types.Task {
	return Queue.GetTask()
}

func Len() int {
	return Queue.Len()
}
