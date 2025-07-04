package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	config "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/rs/xid"
)

type Webdav struct {
	config config.WebdavStorageConfig
	client *Client
	logger *log.Logger
}

func (w *Webdav) Init(ctx context.Context, cfg config.StorageConfig) error {
	webdavConfig, ok := cfg.(*config.WebdavStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast webdav config")
	}
	if err := webdavConfig.Validate(); err != nil {
		return err
	}
	w.config = *webdavConfig
	w.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("webdav[%s]", w.config.Name))
	w.client = NewClient(w.config.URL, w.config.Username, w.config.Password, &http.Client{
		Timeout: time.Hour * 12,
	})
	return nil
}

func (w *Webdav) Type() storenum.StorageType {
	return storenum.Webdav
}

func (w *Webdav) Name() string {
	return w.config.Name
}

func (w *Webdav) JoinStoragePath(p string) string {
	return path.Join(w.config.BasePath, p)
}

func (w *Webdav) Save(ctx context.Context, r io.Reader, storagePath string) error {
	w.logger.Infof("Saving file to %s", storagePath)

	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath
	for i := 1; w.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
		if i > 1000 {
			w.logger.Errorf("Too many attempts to find a unique filename for %s", storagePath)
			candidate = fmt.Sprintf("%s_%s%s", base, xid.New().String(), ext)
			break
		}
	}

	if err := w.client.MkDir(ctx, path.Dir(candidate)); err != nil {
		w.logger.Errorf("Failed to create directory %s: %v", path.Dir(candidate), err)
		return ErrFailedToCreateDirectory
	}
	if err := w.client.WriteFile(ctx, candidate, r); err != nil {
		w.logger.Errorf("Failed to write file %s: %v", candidate, err)
		return ErrFailedToWriteFile
	}
	return nil
}

func (w *Webdav) Exists(ctx context.Context, storagePath string) bool {
	w.logger.Debugf("Checking if file exists at %s", storagePath)
	exists, err := w.client.Exists(ctx, storagePath)
	if err != nil {
		w.logger.Errorf("Failed to check if file exists at %s: %v", storagePath, err)
		return false
	}
	return exists
}
