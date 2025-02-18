package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/krau/SaveAny-Bot/storage/alist"
	"github.com/krau/SaveAny-Bot/storage/local"
	"github.com/krau/SaveAny-Bot/storage/webdav"
	"github.com/krau/SaveAny-Bot/types"
)

type Storage interface {
	Init(model types.StorageModel) error
	Type() types.StorageType
	JoinStoragePath(task types.Task) string
	Save(cttx context.Context, localFilePath, storagePath string) error
}

var (
	ErrInvalidStorageID = errors.New("invalid storage ID")
)

var Storages = make(map[uint]Storage)

// Get storage from model, if it exists, otherwise create and init a new storage
func GetStorageFromModel(model types.StorageModel) (Storage, error) {
	if model.ID == 0 {
		return nil, ErrInvalidStorageID
	}
	if storage, ok := Storages[model.ID]; ok {
		return storage, nil
	}
	storage, err := NewStorage(model)
	if err != nil {
		return nil, err
	}
	Storages[model.ID] = storage
	return storage, nil
}

type StorageConstructor func() Storage

var storageConstructors = map[string]StorageConstructor{
	string(types.StorageTypeAlist):  func() Storage { return new(alist.Alist) },
	string(types.StorageTypeLocal):  func() Storage { return new(local.Local) },
	string(types.StorageTypeWebdav): func() Storage { return new(webdav.Webdav) },
}

func NewStorage(model types.StorageModel) (Storage, error) {
	constructor, ok := storageConstructors[model.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported storage type: %s", model.Type)
	}

	storage := constructor()
	if err := storage.Init(model); err != nil {
		return nil, fmt.Errorf("failed to init %s storage: %w", model.Type, err)
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
