package storage

import (
	"context"
	"fmt"
	"io"

	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/storage/alist"
	"github.com/krau/SaveAny-Bot/storage/local"
	"github.com/krau/SaveAny-Bot/storage/minio"
	"github.com/krau/SaveAny-Bot/storage/telegram"
	"github.com/krau/SaveAny-Bot/storage/webdav"
)

type Storage interface {
	Init(ctx context.Context, cfg storcfg.StorageConfig) error
	Type() storenum.StorageType
	Name() string
	JoinStoragePath(p string) string
	Save(ctx context.Context, reader io.Reader, storagePath string) error
	Exists(ctx context.Context, storagePath string) bool
}

type StorageCannotStream interface {
	Storage
	CannotStream() string
}

var Storages = make(map[string]Storage)

type StorageConstructor func() Storage

var storageConstructors = map[storenum.StorageType]StorageConstructor{
	storenum.Alist:    func() Storage { return new(alist.Alist) },
	storenum.Local:    func() Storage { return new(local.Local) },
	storenum.Webdav:   func() Storage { return new(webdav.Webdav) },
	storenum.Minio:    func() Storage { return new(minio.Minio) },
	storenum.Telegram: func() Storage { return new(telegram.Telegram) },
}

func NewStorage(ctx context.Context, cfg storcfg.StorageConfig) (Storage, error) {
	constructor, ok := storageConstructors[cfg.GetType()]
	if !ok {
		return nil, fmt.Errorf("不支持的存储类型: %s", cfg.GetType())
	}

	storage := constructor()
	if err := storage.Init(ctx, cfg); err != nil {
		return nil, fmt.Errorf("初始化 %s 存储失败: %w", cfg.GetName(), err)
	}

	return storage, nil
}
