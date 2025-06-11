package storage

//go:generate go-enum --values --names --noprefix --flag --nocase

// StorageType
/* ENUM(
local, webdav, alist, minio
) */
type StorageType string
