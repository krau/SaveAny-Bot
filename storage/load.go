package storage

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

var UserStorages = make(map[int64][]Storage)

// GetStorageByName returns storage by name from cache or creates new one
// It should NOT be used to get storage for user, use GetStorageByUserIDAndName instead
func GetStorageByName(ctx context.Context, name string) (Storage, error) {
	if name == "" {
		return nil, ErrStorageNameEmpty
	}

	storage, ok := Storages[name]
	if ok {
		return storage, nil
	}
	cfg := config.C().GetStorageByName(name)
	if cfg == nil {
		return nil, fmt.Errorf("未找到存储 %s", name)
	}

	storage, err := NewStorage(ctx, cfg)
	if err != nil {
		return nil, err
	}
	Storages[name] = storage
	return storage, nil
}

// 检查 user 是否可用指定的 storage, 若不可用则返回未找到错误
func GetStorageByUserIDAndName(ctx context.Context, chatID int64, name string) (Storage, error) {
	if name == "" {
		return nil, ErrStorageNameEmpty
	}

	if !config.C().HasStorage(chatID, name) {
		return nil, fmt.Errorf("no storage %s for user %d", name, chatID)
	}

	return GetStorageByName(ctx, name)
}

func GetUserStorages(ctx context.Context, chatID int64) []Storage {
	if chatID <= 0 {
		return nil
	}
	if storages, ok := UserStorages[chatID]; ok {
		return storages
	}
	var storages []Storage
	for _, name := range config.C().GetStorageNamesByUserID(chatID) {
		storage, err := GetStorageByName(ctx, name)
		if err != nil {
			continue
		}
		storages = append(storages, storage)
	}
	return storages
}

func LoadStorages(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Debug("loading storages...")
	for _, storage := range config.C().Storages {
		_, err := GetStorageByName(ctx, storage.GetName())
		if err != nil {
			logger.Errorf("failed to load storage %s: %v", storage.GetName(), err)
		}
	}
	logger.Infof("successfully loaded %d storages", len(Storages))
	for user := range config.C().GetUsersID() {
		UserStorages[int64(user)] = GetUserStorages(ctx, int64(user))
	}
}

// GetTelegramStorageByUserID returns the first enabled Telegram storage for the user
func GetTelegramStorageByUserID(ctx context.Context, chatID int64) (Storage, error) {
	storages := GetUserStorages(ctx, chatID)
	for _, stor := range storages {
		if stor.Type() == storenum.Telegram {
			return stor, nil
		}
	}
	return nil, fmt.Errorf("no telegram storage found for user %d", chatID)
}
