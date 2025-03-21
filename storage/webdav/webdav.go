package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

func (w *Webdav) JoinStoragePath(task types.Task) string {
	return path.Join(w.config.BasePath, task.StoragePath)
}

func (w *Webdav) Save(ctx context.Context, r io.Reader, storagePath string) error {
	common.Log.Infof("Saving file to %s", storagePath)
	if err := w.client.MkDir(ctx, path.Dir(storagePath)); err != nil {
		common.Log.Errorf("Failed to create directory %s: %v", path.Dir(storagePath), err)
		return ErrFailedToCreateDirectory
	}
	if err := w.client.WriteFile(ctx, storagePath, r); err != nil {
		common.Log.Errorf("Failed to write file %s: %v", storagePath, err)
		return ErrFailedToWriteFile
	}
	return nil
}
