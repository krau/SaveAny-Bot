package batchtfile

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
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
	ID           string           `json:"id"`
	Elements     []elementPayload `json:"elements"`
	ChatID       int64            `json:"chat_id"`
	MessageID    int              `json:"message_id"`
	IgnoreErrors bool             `json:"ignore_errors"`
}

type taskCodec struct{}

func init() {
	core.RegisterTaskCodec(tasktype.TaskTypeTgfiles, taskCodec{})
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

func (taskCodec) Unmarshal(data []byte) (core.Executable, error) {
	var p taskPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid task payload: %w", err)
	}
	dler := core.DownloaderClient()
	if dler == nil {
		return nil, fmt.Errorf("no downloader client available")
	}
	elems := make([]TaskElement, 0, len(p.Elements))
	for _, ep := range p.Elements {
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
	return NewBatchTGFileTask(p.ID, context.Background(), elems, progress, p.IgnoreErrors), nil
}
