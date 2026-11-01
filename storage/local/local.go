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
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
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

// resolve maps a storage path to an absolute path inside the storage root. It
// rejects absolute paths and traversal, including existing symlinks.
func (l *Local) resolve(storagePath string) (string, error) {
	clean := filepath.Clean(storagePath)
	if clean == "." {
		clean = ""
	}
	if clean != "" && !filepath.IsLocal(clean) {
		return "", fmt.Errorf("local: storage path escapes base directory: %s", storagePath)
	}
	absPath := filepath.Join(l.config.BasePath, clean)
	realBase, err := filepath.EvalSymlinks(l.config.BasePath)
	if err != nil {
		return absPath, nil
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return absPath, nil
	}
	rel, err := filepath.Rel(realBase, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("local: storage path escapes base directory: %s", storagePath)
	}
	return absPath, nil
}

func (l *Local) Save(ctx context.Context, r io.Reader, storagePath string) error {
	l.logger.Infof("Saving file to %s", storagePath)
	clean := filepath.Clean(storagePath)
	if clean == "." {
		clean = ""
	}
	absPath, err := l.resolve(clean)
	if err != nil {
		return err
	}
	if overwrite, _ := ctx.Value(ctxkey.OverwriteExisting).(bool); !overwrite {
		absPath = fsutil.UniquePath(l.config.BasePath, clean, l.existsPath, 1000)
	}
	if err := fileutil.CreateDir(filepath.Dir(absPath)); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	// Write beside the target and rename, so a failed copy never leaves a
	// truncated file under the final name.
	tempFile, err := os.CreateTemp(filepath.Dir(absPath), "."+filepath.Base(absPath)+".part-*")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := io.Copy(tempFile, r); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0o644); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, absPath); err != nil {
		return err
	}
	removeTemp = false
	return nil
}

func (l *Local) Exists(ctx context.Context, storagePath string) bool {
	absPath, err := l.resolve(storagePath)
	if err != nil {
		return false
	}
	return l.existsPath(absPath)
}

func (l *Local) existsPath(storagePath string) bool {
	absPath, err := filepath.Abs(storagePath)
	if err != nil {
		return false
	}
	return fileutil.IsExist(absPath)
}

// ListFiles implements StorageListable interface
func (l *Local) ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error) {
	absPath, err := l.resolve(dirPath)
	if err != nil {
		return nil, err
	}

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
	absPath, err := l.resolve(filePath)
	if err != nil {
		return nil, 0, err
	}

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
