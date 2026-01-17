package storagetypes

import "time"

// FileInfo 表示文件元数据
type FileInfo struct {
	Name    string
	Path    string
	Size    int64
	IsDir   bool
	ModTime time.Time
}
