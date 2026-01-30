package rclone

import "errors"

var (
	ErrRcloneNotFound    = errors.New("rclone: rclone command not found in PATH")
	ErrRemoteNotFound    = errors.New("rclone: remote not found")
	ErrFailedToSaveFile  = errors.New("rclone: failed to save file")
	ErrFailedToListFiles = errors.New("rclone: failed to list files")
	ErrFailedToOpenFile  = errors.New("rclone: failed to open file")
	ErrFailedToCheckFile = errors.New("rclone: failed to check file exists")
	ErrFailedToCreateDir = errors.New("rclone: failed to create directory")
	ErrCommandFailed     = errors.New("rclone: command execution failed")
)
