package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"golang.org/x/sync/singleflight"
)

var (
	storageMu sync.RWMutex
	// Storages maps storage names to initialized storage instances.
	Storages = make(map[string]Storage)

	userStoragesMu sync.RWMutex
	// UserStorages maps user IDs to their available storage instances.
	UserStorages = make(map[int64][]Storage)

	initFlight singleflight.Group
)

// GetStorage returns the initialized storage instance for name, without
// creating one on demand.
func GetStorage(name string) (Storage, bool) {
	storageMu.RLock()
	defer storageMu.RUnlock()
	s, ok := Storages[name]
	return s, ok
}

// AllStorages returns a snapshot copy of all initialized storages.
func AllStorages() map[string]Storage {
	storageMu.RLock()
	defer storageMu.RUnlock()
	out := make(map[string]Storage, len(Storages))
	for name, s := range Storages {
		out[name] = s
	}
	return out
}

// GetStorageByName returns storage by name from cache or creates new one
// It should NOT be used to get storage for user, use GetStorageByUserIDAndName instead
func GetStorageByName(ctx context.Context, name string) (Storage, error) {
	if name == "" {
		return nil, ErrStorageNameEmpty
	}

	storageMu.RLock()
	storage, ok := Storages[name]
	storageMu.RUnlock()
	if ok {
		return storage, nil
	}
	cfg := config.C().GetStorageByName(name)
	if cfg == nil {
		return nil, fmt.Errorf("storage %s not found", name)
	}

	// Merge concurrent first-time initializations.
	v, err, _ := initFlight.Do("storage:"+name, func() (any, error) {
		storageMu.RLock()
		if existing, ok := Storages[name]; ok {
			storageMu.RUnlock()
			return existing, nil
		}
		storageMu.RUnlock()
		storage, err := NewStorage(ctx, cfg)
		if err != nil {
			return nil, err
		}
		storageMu.Lock()
		defer storageMu.Unlock()
		if existing, ok := Storages[name]; ok {
			return existing, nil
		}
		Storages[name] = storage
		return storage, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(Storage), nil
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
	userStoragesMu.RLock()
	cached, ok := UserStorages[chatID]
	userStoragesMu.RUnlock()
	if ok {
		return cached
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
	storageMu.RLock()
	loaded := len(Storages)
	storageMu.RUnlock()
	logger.Infof("successfully loaded %d storages", loaded)
	for user := range config.C().GetUsersID() {
		uid := int64(user)
		storages := GetUserStorages(ctx, uid)
		userStoragesMu.Lock()
		UserStorages[uid] = storages
		userStoragesMu.Unlock()
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
