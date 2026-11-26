package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/krau/SaveAny-Bot/common/upload"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/taskstatus"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
	"github.com/rs/xid"
)

func TestUploadTaskLifecycle(t *testing.T) {
	ctx := t.Context()
	database.Init(ctx, filepath.Join(t.TempDir(), "tasks.db"))
	t.Cleanup(func() { _ = database.Close() })
	q := initQueue()
	oldCfg := config.C()
	cfg := oldCfg
	cfg.Temp.BasePath = t.TempDir()
	config.Set(cfg)
	t.Cleanup(func() { config.Set(oldCfg) })

	for _, scenario := range []struct {
		name       string
		enqueue    bool
		start      bool
		cancel     bool
		interrupt  bool
		badResults bool
		want       taskstatus.Status
	}{
		{name: "queued cancellation", enqueue: true, cancel: true, want: taskstatus.Cancelled},
		{name: "cancel before recovery enqueues", cancel: true, want: taskstatus.Cancelled},
		{name: "running cancellation", enqueue: true, start: true, cancel: true, want: taskstatus.Cancelled},
		{name: "shutdown interruption", enqueue: true, start: true, interrupt: true, want: taskstatus.Queued},
		{name: "persist then clean", enqueue: true, start: true, want: taskstatus.Completed},
		{name: "persistence failure retains payload", enqueue: true, start: true, badResults: true, want: taskstatus.Running},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			id := xid.New().String()
			stagingRoot := filepath.Join(t.TempDir(), "web-upload")
			payload, err := upload.StagingPayloadPath(stagingRoot, id)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(payload), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(payload, []byte("payload"), 0o600); err != nil {
				t.Fatal(err)
			}
			params, err := json.Marshal(taskspec.Upload{StagingRoot: stagingRoot})
			if err != nil {
				t.Fatal(err)
			}
			if err := database.CreateTaskRecord(ctx, &database.TaskRecord{
				TaskID: id, Type: "upload", Status: "queued", Params: string(params),
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				q.Done(id)
				_ = database.DeleteTaskRecord(ctx, id)
			})

			taskCtx, stop := context.WithCancel(t.Context())
			defer stop()
			entered := make(chan struct{})
			finished := make(chan struct{})
			published := false
			task := &uploadLifecycleTask{
				id: id,
				run: func(ctx context.Context) error {
					close(entered)
					if scenario.cancel || scenario.interrupt {
						<-ctx.Done()
					}
					if scenario.cancel {
						if _, err := os.Stat(payload); err != nil {
							t.Errorf("running task lost its staged payload: %v", err)
						}
						_ = storagetypes.Publish(ctx, func() error { published = true; return nil })
						// A backend that returns nil after cancellation must not mark the task completed.
						return nil
					}
					return ctx.Err()
				},
				items: []ItemResult{{Index: 1, Path: "actual.txt", Status: ItemStatusCompleted}},
			}
			if scenario.badResults {
				task.items = append(task.items, task.items[0])
			}
			if scenario.want == taskstatus.Completed {
				taskCtx = taskevent.WithSink(taskCtx, taskevent.SinkFunc(func(event taskevent.Event) {
					if event.Phase != taskevent.PhaseDone {
						return
					}
					record, err := database.GetTaskRecord(ctx, id)
					if err != nil || record.Status != "completed" || record.Path != event.Path || event.Path != "actual.txt" {
						t.Errorf("completed event before results were persisted: record=%+v event=%+v err=%v", record, event, err)
					}
				}))
			}
			if scenario.enqueue {
				if err := q.Add(queue.NewTask[Executable](taskCtx, id, id, task)); err != nil {
					t.Fatal(err)
				}
			}
			if scenario.start {
				qtask, err := q.Get()
				if err != nil {
					t.Fatal(err)
				}
				go func() {
					defer close(finished)
					runTask(ctx, qtask)
					q.Done(id)
				}()
				select {
				case <-entered:
				case <-finished:
					select {
					case <-entered:
					default:
						t.Fatal("worker stopped before executing the task")
					}
				}
				defer func() { stop(); <-finished }()
			}
			if scenario.cancel {
				if err := CancelTask(ctx, id); err != nil {
					t.Fatal(err)
				}
			}
			if scenario.interrupt {
				stop()
			}
			if scenario.start {
				<-finished
			}
			record, err := database.GetTaskRecord(ctx, id)
			if err != nil || record.Status != scenario.want.String() {
				t.Fatalf("record = %+v, %v, want %s", record, err, scenario.want)
			}
			if published {
				t.Fatal("cancelled task published a file")
			}
			_, err = os.Stat(payload)
			if scenario.want.IsTerminal() && !errors.Is(err, os.ErrNotExist) || !scenario.want.IsTerminal() && err != nil {
				t.Fatalf("staged file stat = %v for %s task", err, scenario.want)
			}
			if scenario.start && scenario.cancel {
				items, err := database.ListTaskItems(ctx, id)
				if err != nil || len(items) != 1 || items[0].Status != "cancelled" {
					t.Fatalf("cancelled task items = %+v, %v", items, err)
				}
			}
		})
	}
}

type uploadLifecycleTask struct {
	id    string
	run   func(context.Context) error
	items []ItemResult
}

func (*uploadLifecycleTask) Type() tasktype.TaskType { return tasktype.TaskTypeUpload }
func (*uploadLifecycleTask) Title() string           { return "task" }
func (task *uploadLifecycleTask) TaskID() string     { return task.id }
func (task *uploadLifecycleTask) Execute(ctx context.Context) error {
	return task.run(ctx)
}
func (task *uploadLifecycleTask) ItemResults() []ItemResult { return task.items }
