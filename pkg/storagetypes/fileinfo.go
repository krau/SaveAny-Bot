package storagetypes

import "time"

// FileInfo represents file metadata
type FileInfo struct {
	Name    string
	Path    string
	Size    int64
	IsDir   bool
	ModTime time.Time
}
