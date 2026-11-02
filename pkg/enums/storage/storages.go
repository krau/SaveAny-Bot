package storage

//go:generate go-enum --values --names --noprefix --flag --nocase

// StorageType
/* ENUM(
local, webdav, alist, telegram, s3, rclone
) */
type StorageType string
