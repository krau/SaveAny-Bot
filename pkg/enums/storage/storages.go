package storage

//go:generate go-enum --values --names --noprefix --flag --nocase

// StorageType
/* ENUM(
local, webdav, alist, minio, telegram
) */
type StorageType string
