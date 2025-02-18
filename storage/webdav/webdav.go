package webdav

import (
	"context"
	"encoding/json"
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
	config config.WebdavConfig
}

var (
	Client *gowebdav.Client
)

func (w *Webdav) Init(model types.StorageModel) error {
	var webdavConfig config.WebdavConfig
	if err := json.Unmarshal([]byte(model.Config), &webdavConfig); err != nil {
		return fmt.Errorf("failed to unmarshal webdav config: %w", err)
	}
	w.config = webdavConfig
	Client = gowebdav.NewClient(webdavConfig.URL, webdavConfig.Username, webdavConfig.Password)
	if err := Client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to webdav server: %w", err)
	}
	Client.SetTimeout(12 * time.Hour)
	return nil
}

func (w *Webdav) Type() types.StorageType {
	return types.StorageTypeWebdav
}

func (w *Webdav) Save(ctx context.Context, filePath, storagePath string) error {
	if err := Client.MkdirAll(path.Dir(storagePath), os.ModePerm); err != nil {
		logger.L.Errorf("Failed to create directory %s: %v", path.Dir(storagePath), err)
		return ErrFailedToCreateDirectory
	}
	file, err := os.Open(filePath)
	if err != nil {
		logger.L.Errorf("Failed to open file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	if err := Client.WriteStream(storagePath, file, os.ModePerm); err != nil {
		logger.L.Errorf("Failed to write file %s: %v", storagePath, err)
		return ErrFailedToWriteFile
	}
	return nil
}

func (w *Webdav) JoinStoragePath(task types.Task) string {
	return path.Join(w.config.BasePath, task.StoragePath)
}
