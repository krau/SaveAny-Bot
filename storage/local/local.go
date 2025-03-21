package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/krau/SaveAny-Bot/common"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/types"
)

type Local struct {
	config config.LocalStorageConfig
}

func (l *Local) Init(cfg config.StorageConfig) error {
	localConfig, ok := cfg.(*config.LocalStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast local config")
	}
	if err := localConfig.Validate(); err != nil {
		return err
	}
	l.config = *localConfig
	err := os.MkdirAll(localConfig.BasePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create local storage directory: %w", err)
	}
	return nil
}

func (l *Local) Type() types.StorageType {
	return types.StorageTypeLocal
}

func (l *Local) Name() string {
	return l.config.Name
}

func (l *Local) JoinStoragePath(task types.Task) string {
	return filepath.Join(l.config.BasePath, task.StoragePath)
}

func (l *Local) Save(ctx context.Context, r io.Reader, storagePath string) error {
	common.Log.Infof("Saving file to %s", storagePath)

	absPath, err := filepath.Abs(storagePath)
	if err != nil {
		return err
	}
	if err := fileutil.CreateDir(filepath.Dir(absPath)); err != nil {
		return err
	}
	file, err := os.Create(absPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, r)
	return err
}
