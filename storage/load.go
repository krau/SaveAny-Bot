package storage

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
)

var UserStorages = make(map[int64][]Storage)

// GetStorageByName returns storage by name from cache or creates new one
func getStorageByName(ctx context.Context, name string) (Storage, error) {
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
		return nil, fmt.Errorf("没有找到用户 %d 的存储 %s", chatID, name)
	}

	return getStorageByName(ctx, name)
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
		storage, err := getStorageByName(ctx, name)
		if err != nil {
			continue
		}
		storages = append(storages, storage)
	}
	return storages
}

func LoadStorages(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Info("加载存储...")
	for _, storage := range config.C().Storages {
		_, err := getStorageByName(ctx, storage.GetName())
		if err != nil {
			logger.Errorf("加载存储 %s 失败: %v", storage.GetName(), err)
		}
	}
	logger.Infof("成功加载 %d 个存储", len(Storages))
	for user := range config.C().GetUsersID() {
		UserStorages[int64(user)] = GetUserStorages(ctx, int64(user))
	}
}
