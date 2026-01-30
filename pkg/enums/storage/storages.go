package storage

//go:generate go-enum --values --names --noprefix --flag --nocase

// StorageType
/* ENUM(
local, webdav, alist, minio, telegram, s3, rclone
) */
type StorageType string
