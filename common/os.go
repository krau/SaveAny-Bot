package common

import (
	"os"
	"path/filepath"
	"time"
)

func RmFileAfter(path string, td time.Duration) {
	_, err := os.Stat(path)
	if err != nil {
		Log.Errorf("Failed to create timer for %s: %s", path, err)
		return
	}
	Log.Debugf("Remove file after %s: %s", td, path)
	time.AfterFunc(td, func() {
		if err := os.Remove(path); err != nil {
			Log.Errorf("Failed to remove file %s: %s", path, err)
		}
	})
}

// 删除目录下的所有内容, 但不删除目录本身
func RemoveAllInDir(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return err
		}
	}
	return nil
}
