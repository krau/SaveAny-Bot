package transfer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage"
)

type elementPayload struct {
	ID            string                `json:"id"`
	SourceStorage string                `json:"source_storage"`
	SourcePath    string                `json:"source_path"`
	FileInfo      storagetypes.FileInfo `json:"file_info"`
	TargetStorage string                `json:"target_storage"`
	TargetPath    string                `json:"target_path"`
}

type taskPayload struct {
	ID           string           `json:"id"`
	Elements     []elementPayload `json:"elements"`
	ChatID       int64            `json:"chat_id"`
	MessageID    int              `json:"message_id"`
	IgnoreErrors bool             `json:"ignore_errors"`
}

type taskCodec struct{}

func init() {
	core.RegisterTaskCodec(tasktype.TaskTypeTransfer, taskCodec{})
}

func (taskCodec) Marshal(task core.Executable) ([]byte, error) {
	t, ok := task.(*Task)
	if !ok {
		return nil, fmt.Errorf("unexpected task type %T", task)
	}
	p := taskPayload{
		ID:           t.ID,
		IgnoreErrors: t.IgnoreErrors,
	}
	for _, elem := range t.elems {
		p.Elements = append(p.Elements, elementPayload{
			ID:            elem.ID,
			SourceStorage: elem.SourceStorage.Name(),
			SourcePath:    elem.SourcePath,
			FileInfo:      elem.FileInfo,
			TargetStorage: elem.TargetStorage.Name(),
			TargetPath:    elem.TargetPath,
		})
	}
	if progress, ok := t.Progress.(*Progress); ok {
		p.ChatID = progress.ChatID
		p.MessageID = progress.MessageID
	}
	return json.Marshal(p)
}

func (taskCodec) Unmarshal(data []byte) (core.Executable, error) {
	var p taskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid task payload: %w", err)
	}
	elems := make([]TaskElement, 0, len(p.Elements))
	for _, ep := range p.Elements {
		source, err := storage.GetStorageByName(context.Background(), ep.SourceStorage)
		if err != nil {
			return nil, fmt.Errorf("source storage %q: %w", ep.SourceStorage, err)
		}
		target, err := storage.GetStorageByName(context.Background(), ep.TargetStorage)
		if err != nil {
			return nil, fmt.Errorf("target storage %q: %w", ep.TargetStorage, err)
		}
		elems = append(elems, TaskElement{
			ID:            ep.ID,
			SourceStorage: source,
			SourcePath:    ep.SourcePath,
			FileInfo:      ep.FileInfo,
			TargetStorage: target,
			TargetPath:    ep.TargetPath,
		})
	}
	var progress ProgressTracker
	if p.ChatID != 0 {
		progress = NewProgressTracker(p.MessageID, p.ChatID)
	}
	return NewTransferTask(p.ID, context.Background(), elems, progress, p.IgnoreErrors), nil
}
