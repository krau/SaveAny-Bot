package fsutil

import (
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
)

// 删除文件夹内的所有文件和子目录, 但不删除文件夹本身
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

func DetectFileExt(fp string) string {
	mt, err := mimetype.DetectFile(fp)
	if err != nil {
		return ""
	}
	return mt.Extension()
}

// 创建文件, 同时路径上创建不存在的目录
func Create(fp string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(fp), os.ModePerm); err != nil {
		return nil, err
	}
	file, err := os.Create(fp)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func RemoveEmptyDirs(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		err := os.Remove(dirPath)
		if err != nil {
			return err
		}
		return RemoveEmptyDirs(filepath.Dir(dirPath))
	}
	return nil
}

func Remove(fp string) error {
	if err := os.Remove(fp); err != nil {
		return err
	}
	return RemoveEmptyDirs(filepath.Dir(fp))
}
