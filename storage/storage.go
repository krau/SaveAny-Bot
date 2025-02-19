package storage

import (
	"context"
	"fmt"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/storage/alist"
	"github.com/krau/SaveAny-Bot/storage/local"
	"github.com/krau/SaveAny-Bot/storage/webdav"
	"github.com/krau/SaveAny-Bot/types"
)

type Storage interface {
	Init(cfg config.StorageConfig) error
	Type() types.StorageType
	Name() string
	JoinStoragePath(task types.Task) string
	Save(cttx context.Context, localFilePath, storagePath string) error
}

var Storages = make(map[string]Storage)

// GetStorageByName returns storage by name from cache or creates new one
func GetStorageByName(name string) (Storage, error) {
	if name == "" {
		return nil, fmt.Errorf("storage name is required")
	}

	storage, ok := Storages[name]
	if ok {
		return storage, nil
	}
	cfg := config.Cfg.GetStorageByName(name)
	if cfg == nil {
		return nil, fmt.Errorf("storage %s not found", name)
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return nil, err
	}
	Storages[name] = storage
	return storage, nil
}

func GetUserStorages(chatID int64) []Storage {
	var storages []Storage
	for _, name := range config.Cfg.GetStorageNamesByUserID(chatID) {
		storage, err := GetStorageByName(name)
		if err != nil {
			continue
		}
		storages = append(storages, storage)
	}
	return storages
}

type StorageConstructor func() Storage

var storageConstructors = map[string]StorageConstructor{
	string(types.StorageTypeAlist):  func() Storage { return new(alist.Alist) },
	string(types.StorageTypeLocal):  func() Storage { return new(local.Local) },
	string(types.StorageTypeWebdav): func() Storage { return new(webdav.Webdav) },
}

func NewStorage(cfg config.StorageConfig) (Storage, error) {
	constructor, ok := storageConstructors[string(cfg.GetType())]
	if !ok {
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.GetType())
	}

	storage := constructor()
	if err := storage.Init(cfg); err != nil {
		return nil, fmt.Errorf("failed to init %s storage: %w", cfg.GetName(), err)
	}

	return storage, nil
}

func GetStorageConfigurableItems(storageType types.StorageType) []string {
	switch storageType {
	case types.StorageTypeAlist:
		return alist.ConfigurableItems
	case types.StorageTypeLocal:
		return local.ConfigurableItems
	case types.StorageTypeWebdav:
		return webdav.ConfigurableItems
	}
	return nil
}
