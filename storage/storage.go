package storage

import (
	"context"
	"errors"

	"github.com/krau/SaveAny-Bot/dao"
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

// LoadExistingStorages loads existing storages from the database, and initializes them
//
// Should only be called at startup
func LoadExistingStorages() error {
	storageModels, err := dao.GetActiveStorages()
	if err != nil {
		return err
	}
	for _, storageModel := range storageModels {
		storage, err := NewStorage(storageModel)
		if err != nil {
			return err
		}
		Storages[storageModel.ID] = storage
	}
	return nil
}

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

func NewStorage(storageModel types.StorageModel) (Storage, error) {
	switch storageModel.Type {
	case string(types.StorageTypeAlist):
		alistStorage := new(alist.Alist)
		if err := alistStorage.Init(storageModel); err != nil {
			return nil, err
		}
		return alistStorage, nil
	case string(types.StorageTypeLocal):
		localStorage := new(local.Local)
		if err := localStorage.Init(storageModel); err != nil {
			return nil, err
		}
		return localStorage, nil
	case string(types.StorageTypeWebdav):
		webdavStorage := new(webdav.Webdav)
		if err := webdavStorage.Init(storageModel); err != nil {
			return nil, err
		}
		return webdavStorage, nil
	}
	return nil, nil
}
