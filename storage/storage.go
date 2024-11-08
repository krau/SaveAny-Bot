package storage

import (
	"context"
	"errors"
	"sync"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage/alist"
	"github.com/krau/SaveAny-Bot/storage/local"
	"github.com/krau/SaveAny-Bot/storage/webdav"
	"github.com/krau/SaveAny-Bot/types"
)

type Storage interface {
	Init()
	Save(cttx context.Context, filePath, storagePath string) error
}

var Storages = make(map[types.StorageType]Storage)
var StorageKeys = make([]types.StorageType, 0)

func Init() {
	logger.L.Debug("Initializing storage...")
	if config.Cfg.Storage.Alist.Enable {
		Storages[types.Alist] = new(alist.Alist)
		Storages[types.Alist].Init()
	}
	if config.Cfg.Storage.Local.Enable {
		Storages[types.Local] = new(local.Local)
		Storages[types.Local].Init()
	}
	if config.Cfg.Storage.Webdav.Enable {
		Storages[types.Webdav] = new(webdav.Webdav)
		Storages[types.Webdav].Init()
	}

	for k := range Storages {
		StorageKeys = append(StorageKeys, k)
	}

	slice.Sort(StorageKeys)

	logger.L.Debug("Storage initialized")
}

func Save(storageType types.StorageType, ctx context.Context, filePath, storagePath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if storageType != types.StorageAll {
		return Storages[storageType].Save(ctx, filePath, storagePath)
	}
	errs := make([]error, 0)
	var wg sync.WaitGroup
	for _, storage := range Storages {
		wg.Add(1)
		go func(storage Storage) {
			defer wg.Done()
			if err := storage.Save(ctx, filePath, storagePath); err != nil {
				errs = append(errs, err)
			}
		}(storage)
	}
	wg.Wait()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
