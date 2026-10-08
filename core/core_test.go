package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

type cancellationTestTask struct {
	execute func(context.Context) error
}

func (cancellationTestTask) Type() tasktype.TaskType             { return tasktype.TaskTypeTgfiles }
func (cancellationTestTask) Title() string                       { return "cancellation test" }
func (cancellationTestTask) TaskID() string                      { return "cancellation-test" }
func (t cancellationTestTask) Execute(ctx context.Context) error { return t.execute(ctx) }

func TestWorkerUsesTaskContextForCancelHook(t *testing.T) {
	engineErr := fmt.Errorf("engine forcibly closed: %w", context.Canceled)
	for _, tt := range []struct {
		name         string
		err          error
		cancelTask   bool
		cancelParent bool
		wantHook     string
	}{
		{"engine closed with active task", engineErr, false, false, "fail"},
		{"task canceled", engineErr, true, false, "cancel"},
		{"task canceled with other error", errors.New("connection EOF"), true, false, "cancel"},
		{"parent canceled", context.Canceled, false, true, "cancel"},
		{"task deadline error", context.DeadlineExceeded, false, false, "fail"},
		{"ordinary failure", errors.New("connection EOF"), false, false, "fail"},
		{"success", nil, false, false, "success"},
		{"success before cancellation", nil, true, false, "success"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			hookPath := filepath.Join(dir, "hook.txt")
			configPath := filepath.Join(dir, "config.toml")
			cfg := "[hook.exec]\n"
			for _, hook := range []string{"cancel", "fail", "success"} {
				command := fmt.Sprintf("echo %s > %q", hook, filepath.ToSlash(hookPath))
				cfg += fmt.Sprintf("task_%s = %q\n", hook, command)
			}
			if err := os.WriteFile(configPath, []byte(cfg), 0600); err != nil {
				t.Fatal(err)
			}
			if err := config.Init(t.Context(), configPath); err != nil {
				t.Fatal(err)
			}
			parentCtx, cancelParent := context.WithCancel(t.Context())
			defer cancelParent()
			var finalEvent taskevent.Event
			var gotFinal bool
			base := taskevent.WithSink(parentCtx, taskevent.SinkFunc(func(e taskevent.Event) {
				if e.Phase == taskevent.PhaseDone {
					finalEvent, gotFinal = e, true
				}
			}))
			var qtask *queue.Task[Executable]
			exe := cancellationTestTask{execute: func(ctx context.Context) error {
				if tt.cancelTask {
					qtask.Cancel()
				}
				if tt.cancelParent {
					cancelParent()
				}
				if !tt.cancelTask && !tt.cancelParent && ctx.Err() != nil {
					t.Fatalf("task context unexpectedly canceled: %v", ctx.Err())
				}
				return tt.err
			}}
			qtask = queue.NewTask[Executable](base, exe.TaskID(), exe.Title(), exe)
			defer qtask.Cancel()
			q := queue.NewTaskQueue[Executable]()
			if err := q.Add(qtask); err != nil {
				t.Fatal(err)
			}
			q.Close()
			worker(t.Context(), q, make(chan struct{}, 1))
			data, err := os.ReadFile(hookPath)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(string(data)); got != tt.wantHook {
				t.Fatalf("hook = %q, want %q", got, tt.wantHook)
			}
			if !gotFinal || finalEvent.Err != tt.err {
				t.Fatalf("final event = %+v, want original error %v", finalEvent, tt.err)
			}
		})
	}
}
