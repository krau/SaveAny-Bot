package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/types"
)

type Local struct {
	config config.LocalConfig
}

func (l *Local) Init(model types.StorageModel) error {
	var localConfig config.LocalConfig
	if err := json.Unmarshal([]byte(model.Config), &localConfig); err != nil {
		return fmt.Errorf("failed to unmarshal local config: %w", err)
	}
	l.config = localConfig
	err := os.MkdirAll(localConfig.BasePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create local storage directory: %w", err)
	}
	return nil
}

func (l *Local) Type() types.StorageType {
	return types.StorageTypeLocal
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

func (l *Local) JoinStoragePath(task types.Task) string {
	return filepath.Join(l.config.BasePath, task.StoragePath)
}
