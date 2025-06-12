package tftask

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/downloader"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
	"github.com/krau/SaveAny-Bot/pkg/enums/key"
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
	client    Client
	stream    bool // true if the file should be downloaded in stream mode
	localPath string
}

func (t *TGFileTask) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", t.File.Name()))
	t.Progress.OnStart(ctx, t)
	if t.stream {
		return executeStream(ctx, t)
	}

	logger.Info("Starting file download")
	localFile, err := os.Create(t.localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	wrAt := newWriterAt(ctx, localFile, t.Progress, t)
	defer func() {
		if err := localFile.Close(); err != nil {
			logger.Errorf("Failed to close local file: %v", err)
		}
	}()

	defer func() {
		t.Progress.OnDone(ctx, t, err)
	}()
	dler := downloader.NewDownloader().WithPartSize(tglimit.MaxPartSize).Download(t.client, t.File.Location())
	_, err = dler.WithThreads(BestThreads(t.File.Size(), config.Cfg.Threads)).Parallel(ctx, wrAt)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Infof("File downloaded successfully")
	if path.Ext(t.File.Name()) == "" {
		ext := fsutil.DetectFileExt(t.localPath)
		if ext != "" {
			t.Path = t.Path + ext
		}
	}
	localFile, err = fsutil.Open(t.localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()
	var fileStat os.FileInfo
	fileStat, err = localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}
	vctx := context.WithValue(ctx, key.ContextKeyContentLength, fileStat.Size())
	for i := range config.Cfg.Retry + 1 {
		if err = vctx.Err(); err != nil {
			return fmt.Errorf("context canceled while saving file: %w", err)
		}
		var file *os.File
		file, err = os.Open(t.localPath)
		if err != nil {
			return fmt.Errorf("failed to open cache file: %w", err)
		}
		defer file.Close()
		if err = t.Storage.Save(vctx, file, t.Path); err != nil {
			if i == config.Cfg.Retry {
				return fmt.Errorf("failed to save file: %w", err)
			}
			logger.Errorf("Failed to save file: %s, retrying...", err)
			select {
			case <-vctx.Done():
				return fmt.Errorf("context canceled during retry delay: %w", vctx.Err())
			case <-time.After(time.Duration(i*500) * time.Millisecond):
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to save file after retries")

}

type Client interface {
	downloader.Client
}

func NewTGFileTask(
	id string,
	ctx context.Context,
	file tfile.TGFile,
	client Client,
	stor storage.Storage,
	path string,
	progress ProgressTracker,
) (*TGFileTask, error) {
	// TODO: STREAM mode
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
