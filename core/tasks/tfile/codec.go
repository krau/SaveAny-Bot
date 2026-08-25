package tfile

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

type taskPayload struct {
	ID        string               `json:"id"`
	Storage   string               `json:"storage"`
	Path      string               `json:"path"`
	File      tfilepkg.FilePayload `json:"file"`
	ChatID    int64                `json:"chat_id"`
	MessageID int                  `json:"message_id"`
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
	filePayload, ok := tfilepkg.FilePayloadOf(t.File)
	if !ok {
		return nil, fmt.Errorf("file %T is not serializable", t.File)
	}
	p := taskPayload{
		ID:      t.ID,
		Storage: t.Storage.Name(),
		Path:    t.Path,
		File:    filePayload,
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
	file := tfilepkg.FileFromPayload(p.File, dler)
	stor, err := storage.GetStorageByName(context.Background(), p.Storage)
	if err != nil {
		return nil, fmt.Errorf("storage %q: %w", p.Storage, err)
	}
	var progress ProgressTracker
	if p.ChatID != 0 {
		progress = NewProgressTrack(p.MessageID, p.ChatID)
	}
	localPath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", p.ID, file.Name())))
	if err != nil {
		return nil, fmt.Errorf("failed to build cache path: %w", err)
	}
	return &Task{
		ID:        p.ID,
		Ctx:       context.Background(),
		File:      file,
		Storage:   stor,
		Path:      p.Path,
		Progress:  progress,
		stream:    false, // recovered tasks always download to cache first
		localPath: localPath,
	}, nil
}
