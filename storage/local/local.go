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
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
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
	storagePath = l.JoinStoragePath(storagePath)

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

// ListFiles implements StorageListable interface
func (l *Local) ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error) {
	absPath := l.JoinStoragePath(dirPath)
	
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", absPath, err)
	}
	
	files := make([]storagetypes.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			l.logger.Warnf("Failed to get file info for %s: %v", entry.Name(), err)
			continue
		}
		
		filePath := filepath.Join(dirPath, entry.Name())
		files = append(files, storagetypes.FileInfo{
			Name:    entry.Name(),
			Path:    filePath,
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		})
	}
	
	return files, nil
}

// OpenFile implements StorageReadable interface
func (l *Local) OpenFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	absPath := l.JoinStoragePath(filePath)
	
	file, err := os.Open(absPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file %s: %w", absPath, err)
	}
	
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("failed to stat file %s: %w", absPath, err)
	}
	
	return file, stat.Size(), nil
}
