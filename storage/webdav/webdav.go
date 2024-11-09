package webdav

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/studio-b12/gowebdav"
)

type Webdav struct{}

var (
	Client   *gowebdav.Client
	basePath string
)

func (w *Webdav) Init() {
	webdavConfig := config.Cfg.Storage.Webdav
	basePath = strings.TrimSuffix(webdavConfig.BasePath, "/")
	Client = gowebdav.NewClient(webdavConfig.URL, webdavConfig.Username, webdavConfig.Password)
	if err := Client.Connect(); err != nil {
		logger.L.Fatalf("Failed to connect to webdav server: %v", err)
		os.Exit(1)
	}
	Client.SetTimeout(24 * time.Hour)
}

func (w *Webdav) Save(ctx context.Context, filePath, storagePath string) error {
	storagePath = basePath + "/" + storagePath
	if err := Client.MkdirAll(filepath.Dir(storagePath), os.ModePerm); err != nil {
		logger.L.Errorf("Failed to create directory %s: %v", filepath.Dir(storagePath), err)
		return errors.New("webdav: failed to create directory")
	}
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		logger.L.Errorf("Failed to read file %s: %v", filePath, err)
		return err
	}
	if err := Client.Write(storagePath, fileBytes, os.ModePerm); err != nil {
		logger.L.Errorf("Failed to write file %s: %v", storagePath, err)
		return errors.New("webdav: failed to write file")
	}
	return nil
}
