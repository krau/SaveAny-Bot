package fsutil

import (
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
)

// func RmFileAfter(path string, td time.Duration) {
// 	_, err := os.Stat(path)
// 	if err != nil {
// 		Log.Errorf(i18n.T(i18nk.CreateRmTimerFailed, map[string]any{
// 			"Path":  path,
// 			"Error": err,
// 		}))
// 		return
// 	}
// 	Log.Debugf(i18n.T(i18nk.RemoveFileAfter, map[string]any{
// 		"Duration": td.String(),
// 		"Path":     path,
// 	}))
// 	time.AfterFunc(td, func() {
// 		if err := os.Remove(path); err != nil {
// 			Log.Errorf(i18n.T(i18nk.RemoveFileFailed, map[string]any{
// 				"Path":  path,
// 				"Error": err,
// 			}))
// 		}
// 	})
// }

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

func Open(fp string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(fp), os.ModePerm); err != nil {
		return nil, err
	}
	file, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	return file, nil
}
