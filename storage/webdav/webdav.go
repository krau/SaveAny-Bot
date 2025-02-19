package webdav

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/types"
	"github.com/studio-b12/gowebdav"
)

type Webdav struct {
	config config.WebdavStorageConfig
	client *gowebdav.Client
}

var ConfigurableItems = []string{"url", "username", "password", "base_path"}

func (w *Webdav) Init(cfg config.StorageConfig) error {
	webdavConfig, ok := cfg.(*config.WebdavStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast webdav config")
	}
	if err := webdavConfig.Validate(); err != nil {
		return err
	}
	w.config = *webdavConfig
	client := gowebdav.NewClient(webdavConfig.URL, webdavConfig.Username, webdavConfig.Password)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to webdav server: %w", err)
	}
	client.SetTimeout(12 * time.Hour)
	w.client = client
	return nil
}

func (w *Webdav) Type() types.StorageType {
	return types.StorageTypeWebdav
}

func (w *Webdav) Name() string {
	return w.config.Name
}

func (w *Webdav) Save(ctx context.Context, filePath, storagePath string) error {
	if err := w.client.MkdirAll(path.Dir(storagePath), os.ModePerm); err != nil {
		logger.L.Errorf("Failed to create directory %s: %v", path.Dir(storagePath), err)
		return ErrFailedToCreateDirectory
	}
	file, err := os.Open(filePath)
	if err != nil {
		logger.L.Errorf("Failed to open file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	if err := w.client.WriteStream(storagePath, file, os.ModePerm); err != nil {
		logger.L.Errorf("Failed to write file %s: %v", storagePath, err)
		return ErrFailedToWriteFile
	}
	return nil
}

func (w *Webdav) JoinStoragePath(task types.Task) string {
	return path.Join(w.config.BasePath, task.StoragePath)
}
