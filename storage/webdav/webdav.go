package webdav

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/krau/SaveAny-Bot/common"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/types"
)

type Webdav struct {
	config config.WebdavStorageConfig
	client *Client
}

func (w *Webdav) Init(cfg config.StorageConfig) error {
	webdavConfig, ok := cfg.(*config.WebdavStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast webdav config")
	}
	if err := webdavConfig.Validate(); err != nil {
		return err
	}
	w.config = *webdavConfig
	w.client = NewClient(w.config.URL, w.config.Username, w.config.Password, &http.Client{
		Timeout: time.Hour * 12,
	})
	return nil
}

func (w *Webdav) Type() types.StorageType {
	return types.StorageTypeWebdav
}

func (w *Webdav) Name() string {
	return w.config.Name
}

func (w *Webdav) Save(ctx context.Context, filePath, storagePath string) error {
	common.Log.Infof("Saving file %s to %s", filePath, storagePath)
	if err := w.client.MkDir(ctx, path.Dir(storagePath)); err != nil {
		common.Log.Errorf("Failed to create directory %s: %v", path.Dir(storagePath), err)
		return ErrFailedToCreateDirectory
	}
	file, err := os.Open(filePath)
	if err != nil {
		common.Log.Errorf("Failed to open file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	if err := w.client.WriteFile(ctx, storagePath, file); err != nil {
		common.Log.Errorf("Failed to write file %s: %v", storagePath, err)
		return ErrFailedToWriteFile
	}
	return nil
}

func (w *Webdav) JoinStoragePath(task types.Task) string {
	return path.Join(w.config.BasePath, task.StoragePath)
}
