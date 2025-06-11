package webdav

import "errors"

var (
	ErrFailedToCreateDirectory = errors.New("webdav: failed to create directory")
	ErrFailedToWriteFile       = errors.New("webdav: failed to write file")
	ErrFailedToCheckFileExists = errors.New("webdav: failed to check if file exists")
)
