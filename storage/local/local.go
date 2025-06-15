package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/fileutil"
	config "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type Local struct {
	config config.LocalStorageConfig
	logger *log.Logger
}

func (l *Local) Init(ctx context.Context, cfg config.StorageConfig) error {
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
	l.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("local[%s]", l.config.Name))
	return nil
}

func (l *Local) Type() storenum.StorageType {
	return storenum.Local
}

func (l *Local) Name() string {
	return l.config.Name
}

func (l *Local) JoinStoragePath(path string) string {
	return filepath.Join(l.config.BasePath, path)
}

func (l *Local) Save(ctx context.Context, r io.Reader, storagePath string) error {
	l.logger.Infof("Saving file to %s", storagePath)

	ext := filepath.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath
	for i := 1; l.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
	}

	absPath, err := filepath.Abs(candidate)
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

func (l *Local) Exists(ctx context.Context, storagePath string) bool {
	absPath, err := filepath.Abs(storagePath)
	if err != nil {
		return false
	}
	return fileutil.IsExist(absPath)
}
