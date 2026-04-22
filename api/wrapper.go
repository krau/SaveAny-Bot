package api

import (
	"context"
	"errors"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

// ExecutableWrapper wraps core.Executable to track task status in the API store and send webhooks.
type ExecutableWrapper struct {
	inner core.Executable
}

func NewExecutableWrapper(inner core.Executable) *ExecutableWrapper {
	return &ExecutableWrapper{inner: inner}
}

func (w *ExecutableWrapper) Type() tasktype.TaskType { return w.inner.Type() }
func (w *ExecutableWrapper) Title() string           { return w.inner.Title() }
func (w *ExecutableWrapper) TaskID() string          { return w.inner.TaskID() }

func (w *ExecutableWrapper) Execute(ctx context.Context) error {
	taskID := w.inner.TaskID()

	if info, ok := GetTask(taskID); ok {
		info.UpdateStatus(TaskStatusRunning)
	}

	err := w.inner.Execute(ctx)

	info, ok := GetTask(taskID)
	if !ok {
		return err
	}

	var status TaskStatus
	if err != nil {
		if errors.Is(err, context.Canceled) {
			status = TaskStatusCancelled
			info.UpdateStatus(TaskStatusCancelled)
		} else {
			status = TaskStatusFailed
			info.SetError(err.Error())
		}
	} else {
		status = TaskStatusCompleted
		info.UpdateStatus(TaskStatusCompleted)
	}

	if info.Webhook != "" {
		payload := CreateWebhookPayload(taskID, info.Type, status, info.Storage, info.Path, err)
		SendWebhook(ctx, payload)
	}

	return err
}
