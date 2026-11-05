package tfile

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/metadata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.Executable = (*Task)(nil)
var _ core.DestinationInfo = (*Task)(nil)

type Task struct {
	ID              string
	ElementID       string
	Done            []string
	Ctx             context.Context
	File            tfile.TGFile
	Storage         storage.Storage
	Path            string
	Progress        ProgressTracker
	stream          bool
	localPath       string
	execErr         error
	metadataJSON    string
	metadataVersion int
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
	metaJSON, metaVersion := buildFileMetadata(ctx, file)
	_, ok := stor.(storage.StorageCannotStream)
	if !config.C().Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", id, fsutil.SafeBaseName(file.Name()))))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache: %w", err)
		}
		tfile := &Task{
			ID:              id,
			ElementID:       id,
			Ctx:             ctx,
			File:            file,
			Storage:         stor,
			Path:            path,
			Progress:        progress,
			localPath:       cachePath,
			metadataJSON:    metaJSON,
			metadataVersion: metaVersion,
		}
		return tfile, nil
	}
	tfileTask := &Task{
		ID:              id,
		ElementID:       id,
		Ctx:             ctx,
		File:            file,
		Storage:         stor,
		Path:            path,
		Progress:        progress,
		stream:          true,
		metadataJSON:    metaJSON,
		metadataVersion: metaVersion,
	}
	return tfileTask, nil
}

func buildFileMetadata(ctx context.Context, file tfile.TGFile) (string, int) {
	if !config.C().Metadata.Enabled {
		return "", 0
	}
	messageFile, ok := file.(tfile.TGFileMessage)
	if !ok || messageFile.Message() == nil {
		return "", 0
	}
	metaJSON, version, err := metadata.BuildJSON(ctx, messageFile.Message(), file.Name(), file.Size(), config.C().Metadata.IncludeText)
	if err != nil {
		log.FromContext(ctx).Warnf("Failed to build task metadata: %v", err)
		return "", 0
	}
	return metaJSON, version
}

func (t *Task) saveMetadataSidecar(ctx context.Context, actualPath string) {
	if !config.C().Metadata.Sidecar || t.metadataJSON == "" || actualPath == "" {
		return
	}
	if err := storage.WriteMetadataSidecar(ctx, t.Storage, actualPath, []byte(t.metadataJSON)); err != nil {
		log.FromContext(ctx).Warnf("Failed to save metadata sidecar for %s: %v", actualPath, err)
	}
}
