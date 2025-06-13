package tftask

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/krau/SaveAny-Bot/common/tdler"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

type TGFileTask struct {
	ID        string
	Ctx       context.Context
	File      tfile.TGFile
	Storage   storage.Storage
	Path      string
	Progress  ProgressTracker
	client    tdler.Client
	stream    bool // true if the file should be downloaded in stream mode
	localPath string
}

func NewTGFileTask(
	id string,
	ctx context.Context,
	file tfile.TGFile,
	client tdler.Client,
	stor storage.Storage,
	path string,
	progress ProgressTracker,
) (*TGFileTask, error) {
	_, ok := stor.(storage.StorageCannotStream)
	if !config.Cfg.Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.Cfg.Temp.BasePath, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache: %w", err)
		}
		tftask := &TGFileTask{
			ID:        id,
			Ctx:       ctx,
			client:    client,
			File:      file,
			Storage:   stor,
			Path:      path,
			Progress:  progress,
			localPath: cachePath,
		}
		return tftask, nil
	}
	tfileTask := &TGFileTask{
		ID:       id,
		Ctx:      ctx,
		client:   client,
		File:     file,
		Storage:  stor,
		Path:     path,
		Progress: progress,
		stream:   true,
	}
	return tfileTask, nil
}
