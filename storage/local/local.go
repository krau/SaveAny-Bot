package local

import (
	"context"
	"os"
	"path/filepath"

	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
)

type Local struct{}

func (l *Local) Init() {
	err := os.MkdirAll(config.Cfg.Storage.Local.BasePath, os.ModePerm)
	if err != nil {
		logger.L.Fatalf("Failed to create local storage directory: %s", err)
		os.Exit(1)
	}
}

func (l *Local) Save(ctx context.Context, filePath, storagePath string) error {
	absPath, err := filepath.Abs(storagePath)
	if err != nil {
		return err
	}
	if err := fileutil.CreateDir(filepath.Dir(absPath)); err != nil {
		return err
	}
	return fileutil.CopyFile(filePath, storagePath)
}
