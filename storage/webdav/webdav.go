package webdav

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/studio-b12/gowebdav"
)

type Webdav struct{}

var (
	Client *gowebdav.Client
)

func (w *Webdav) Init() {
	webdavConfig := config.Cfg.Storage.Webdav
	Client = gowebdav.NewClient(webdavConfig.URL, webdavConfig.Username, webdavConfig.Password)
	if err := Client.Connect(); err != nil {
		logger.L.Fatalf("Failed to connect to webdav server: %v", err)
		os.Exit(1)
	}
	Client.SetTimeout(24 * time.Hour)
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
