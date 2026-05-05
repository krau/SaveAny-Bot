package tfile

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/pkg/metadata"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.Executable = (*Task)(nil)

type Task struct {
	ID        string
	Ctx       context.Context
	File      tfile.TGFile
	Storage   storage.Storage
	Path      string
	Progress  ProgressTracker
	stream    bool // true if the file should be downloaded in stream mode
	localPath string
	metadata  []byte // pre-built JSON metadata, nil if save_metadata is disabled
}

// Title implements core.Exectable.
func (t *Task) Title() string {
	return fmt.Sprintf("[%s](%s->%s:%s)", t.Type(), t.File.Name(), t.Storage.Name(), t.Path)
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeTgfiles
}

func NewTGFileTask(
	id string,
	ctx context.Context,
	file tfile.TGFile,
	stor storage.Storage,
	path string,
	progress ProgressTracker,
) (*Task, error) {
	var meta []byte
	if config.C().SaveMetadata {
		if fmsg, ok := file.(tfile.TGFileMessage); ok {
			m := metadata.BuildFromMessage(ctx, fmsg.Message(), file.Name(), file.Size())
			var err error
			meta, err = m.ToJSON()
			if err != nil {
				log.FromContext(ctx).Warnf("failed to marshal metadata: %s", err)
			}
		}
	}
	_, ok := stor.(storage.StorageCannotStream)
	if !config.C().Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", id, file.Name())))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache: %w", err)
		}
		return &Task{
			ID:        id,
			Ctx:       ctx,
			File:      file,
			Storage:   stor,
			Path:      path,
			Progress:  progress,
			localPath: cachePath,
			metadata:  meta,
		}, nil
	}
	return &Task{
		ID:       id,
		Ctx:      ctx,
		File:     file,
		Storage:  stor,
		Path:     path,
		Progress: progress,
		stream:   true,
		metadata: meta,
	}, nil
}

func (t *Task) saveMetadata(ctx context.Context, actualPath string) error {
	if len(t.metadata) == 0 {
		return nil
	}
	_, err := t.Storage.Save(ctx, bytes.NewReader(t.metadata), actualPath+metadata.MetaSuffix)
	return err
}
