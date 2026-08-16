package batchtfile

import (
	"context"
	"sync/atomic"
	"testing"
)

type recordingTracker struct {
	calls atomic.Int64
}

func (r *recordingTracker) OnStart(context.Context, TaskInfo)  {}
func (r *recordingTracker) OnDone(context.Context, TaskInfo, error) {
}
func (r *recordingTracker) OnProgress(context.Context, TaskInfo) {
	r.calls.Add(1)
}

// Regression: notifyProgress must call the tracker, not itself. The previous
// self-call recursed until stack overflow on any batch task with a tracker.
func TestNotifyProgressCallsTracker(t *testing.T) {
	tracker := &recordingTracker{}
	task := &Task{Progress: tracker}
	task.notifyProgress(t.Context())
	if tracker.calls.Load() != 1 {
		t.Fatalf("expected 1 tracker call, got %d", tracker.calls.Load())
	}
}

// The nil tracker path must stay a no-op.
func TestNotifyProgressNilTracker(t *testing.T) {
	task := &Task{}
	task.notifyProgress(t.Context()) // must not panic
}
