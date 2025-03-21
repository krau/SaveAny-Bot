package common

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// 创建文件, 自动创建目录
func MkFile(path string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, os.ModePerm)
}

// 删除文件, 并清理空目录. 如果文件不存在则返回 nil
func PurgeFile(path string) error {
	if err := os.Remove(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return RemoveEmptyDirectories(filepath.Dir(path))
}

func RmFileAfter(path string, td time.Duration) {
	_, err := os.Stat(path)
	if err != nil {
		Log.Errorf("Failed to create timer for %s: %s", path, err)
		return
	}
	Log.Debugf("Remove file after %s: %s", td, path)
	time.AfterFunc(td, func() {
		PurgeFile(path)
	})
}

// 递归删除空目录
func RemoveEmptyDirectories(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		err := os.Remove(dirPath)
		if err != nil {
			return err
		}
		return RemoveEmptyDirectories(filepath.Dir(dirPath))
	}
	return nil
}
