package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

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

type File struct {
	*os.File
}

func (f *File) Remove() error {
	return os.Remove(f.Name())
}

func (f *File) CloseAndRemove() error {
	if err := f.Close(); err != nil {
		return err
	}
	return f.Remove()
}

func CreateFile(fp string) (*File, error) {
	if err := os.MkdirAll(filepath.Dir(fp), os.ModePerm); err != nil {
		return nil, err
	}
	file, err := os.Create(fp)
	if err != nil {
		return nil, err
	}
	return &File{File: file}, nil
}

func NormalizePathname(s string) string {
	specials := `\/:*?"<>|` + "\n\r\t"
	var builder strings.Builder
	for _, ch := range s {
		if strings.ContainsRune(specials, ch) || unicode.IsControl(ch) {
			builder.WriteRune('_')
		} else {
			builder.WriteRune(ch)
		}
	}

	result := strings.TrimRightFunc(builder.String(), func(r rune) bool {
		return r == '.' || r == '_' || unicode.IsSpace(r)
	})

	return result
}
