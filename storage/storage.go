package storage

import (
	"context"
	"fmt"
	"io"

	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage/alist"
	"github.com/krau/SaveAny-Bot/storage/local"
	"github.com/krau/SaveAny-Bot/storage/minio"
	"github.com/krau/SaveAny-Bot/storage/rclone"
	"github.com/krau/SaveAny-Bot/storage/s3"
	"github.com/krau/SaveAny-Bot/storage/telegram"
	"github.com/krau/SaveAny-Bot/storage/webdav"
)

type Storage interface {
	// Init 只应该在创建存储时调用一次
	Init(ctx context.Context, cfg storcfg.StorageConfig) error
	Type() storenum.StorageType
	Name() string
	Save(ctx context.Context, reader io.Reader, storagePath string) error
	Exists(ctx context.Context, storagePath string) bool
}

type StorageCannotStream interface {
	Storage
	CannotStream() string
}

// StorageListable 表示支持列举目录内容的存储
type StorageListable interface {
	Storage
	ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error)
}

// StorageReadable 表示支持读取文件内容的存储
type StorageReadable interface {
	Storage
	OpenFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error)
}

var Storages = make(map[string]Storage)

type StorageConstructor func() Storage

var storageConstructors = map[storenum.StorageType]StorageConstructor{
	storenum.Alist:    func() Storage { return new(alist.Alist) },
	storenum.Local:    func() Storage { return new(local.Local) },
	storenum.Webdav:   func() Storage { return new(webdav.Webdav) },
	storenum.Minio:    func() Storage { return new(minio.Minio) },
	storenum.S3:       func() Storage { return new(s3.S3) },
	storenum.Telegram: func() Storage { return new(telegram.Telegram) },
	storenum.Rclone:   func() Storage { return new(rclone.Rclone) },
}

// NewStorage creates a new storage instance based on the provided config and initializes it
func NewStorage(ctx context.Context, cfg storcfg.StorageConfig) (Storage, error) {
	constructor, ok := storageConstructors[cfg.GetType()]
	if !ok {
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.GetType())
	}

	storage := constructor()
	if err := storage.Init(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize storage %s: %w", cfg.GetName(), err)
	}

	return storage, nil
}
