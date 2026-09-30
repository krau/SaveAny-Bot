package batchtfile

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

type elementPayload struct {
	ID              string               `json:"id"`
	Storage         string               `json:"storage"`
	Path            string               `json:"path"`
	File            tfilepkg.FilePayload `json:"file"`
	SourceGroupKey  string               `json:"source_group_key"`
	SourceCaption   string               `json:"source_caption"`
	PreserveCaption bool                 `json:"preserve_caption"`
}

type taskPayload struct {
	Kind         string           `json:"kind"` // "batch"
	ID           string           `json:"id"`
	Elements     []elementPayload `json:"elements"`
	ChatID       int64            `json:"chat_id"`
	MessageID    int              `json:"message_id"`
	IgnoreErrors bool             `json:"ignore_errors"`
	Overwrite    bool             `json:"overwrite"`
	// Done lists element IDs whose upload completed; they are skipped on recovery.
	Done []string `json:"done"`
}

// tgfilesCodec is the single codec registered for TaskTypeTgfiles: it
// dispatches between single-file and batch tasks by concrete type on marshal
// and by payload shape on unmarshal. Registering one codec per task class
// under the shared TaskTypeTgfiles key would let the last init() win and
// silently disable persistence for the other class.
type tgfilesCodec struct{}

func init() {
	core.RegisterTaskCodec(tasktype.TaskTypeTgfiles, tgfilesCodec{})
}

func (tgfilesCodec) Marshal(task core.Executable) ([]byte, error) {
	switch t := task.(type) {
	case *tftask.Task:
		return tftask.TaskCodec.Marshal(t)
	case *Task:
		return batchCodec{}.Marshal(t)
	default:
		return nil, fmt.Errorf("unexpected task type %T", task)
	}
}

// detectTaskKind returns "batch" or "file" for a persisted tgfiles payload.
// New payloads carry an explicit kind; legacy payloads are detected by shape.
func detectTaskKind(data []byte) (string, error) {
	var shape struct {
		Kind     string            `json:"kind"`
		Elements []json.RawMessage `json:"elements"`
		File     json.RawMessage   `json:"file"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return "", fmt.Errorf("invalid task payload: %w", err)
	}
	switch {
	case shape.Kind == "batch", shape.Kind == "" && shape.Elements != nil:
		return "batch", nil
	case shape.Kind == "file", shape.Kind == "" && shape.File != nil:
		return "file", nil
	default:
		return "", fmt.Errorf("unrecognized task payload")
	}
}

func (tgfilesCodec) Unmarshal(data []byte) (core.Executable, error) {
	kind, err := detectTaskKind(data)
	if err != nil {
		return nil, err
	}
	if kind == "batch" {
		return batchCodec{}.Unmarshal(data)
	}
	return tftask.TaskCodec.Unmarshal(data)
}

type batchCodec struct{}

func (batchCodec) Marshal(task core.Executable) ([]byte, error) {
	t, ok := task.(*Task)
	if !ok {
		return nil, fmt.Errorf("unexpected task type %T", task)
	}
	p := taskPayload{
		Kind:         "batch",
		ID:           t.ID,
		IgnoreErrors: t.IgnoreErrors,
		Done:         t.completedElementIDs(),
	}
	if overwrite, ok := t.ctx.Value(ctxkey.OverwriteExisting).(bool); ok {
		p.Overwrite = overwrite
	}
	// 持久化时合并已记录的完成列表, 防止丢失。
	if t.overwrite {
		p.Overwrite = true
	}
	p.Done = append(t.doneList, p.Done...)
	for _, elem := range t.elems {
		filePayload, ok := tfilepkg.FilePayloadOf(elem.File)
		if !ok {
			return nil, fmt.Errorf("file %T is not serializable", elem.File)
		}
		p.Elements = append(p.Elements, elementPayload{
			ID:              elem.ID,
			Storage:         elem.Storage.Name(),
			Path:            elem.Path,
			File:            filePayload,
			SourceGroupKey:  elem.sourceGroupKey,
			SourceCaption:   elem.sourceCaption,
			PreserveCaption: elem.preserveCaption,
		})
	}
	if progress, ok := t.Progress.(*Progress); ok {
		p.ChatID = progress.ChatID
		p.MessageID = progress.MessageID
	}
	return json.Marshal(p)
}

func (batchCodec) Unmarshal(data []byte) (core.Executable, error) {
	var p taskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid task payload: %w", err)
	}
	dler := core.DownloaderClient()
	if dler == nil {
		return nil, fmt.Errorf("no downloader client available")
	}
	done := make(map[string]struct{}, len(p.Done))
	for _, id := range p.Done {
		done[id] = struct{}{}
	}
	elems := make([]TaskElement, 0, len(p.Elements))
	for _, ep := range p.Elements {
		if _, ok := done[ep.ID]; ok {
			continue // upload already completed; do not re-run
		}
		stor, err := storage.GetStorageByName(context.Background(), ep.Storage)
		if err != nil {
			return nil, fmt.Errorf("storage %q: %w", ep.Storage, err)
		}
		localPath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", ep.ID, ep.File.Name)))
		if err != nil {
			return nil, fmt.Errorf("failed to build cache path: %w", err)
		}
		elems = append(elems, TaskElement{
			ID:              ep.ID,
			Storage:         stor,
			Path:            ep.Path,
			File:            tfilepkg.FileFromPayload(ep.File, dler),
			localPath:       localPath,
			sourceGroupKey:  ep.SourceGroupKey,
			sourceCaption:   ep.SourceCaption,
			preserveCaption: ep.PreserveCaption,
		})
	}
	var progress ProgressTracker
	if p.ChatID != 0 {
		progress = NewProgressTracker(p.MessageID, p.ChatID)
	}
	task := NewBatchTGFileTask(p.ID, context.Background(), elems, progress, p.IgnoreErrors)
	task.overwrite = p.Overwrite
	task.doneList = p.Done
	return task, nil
}
