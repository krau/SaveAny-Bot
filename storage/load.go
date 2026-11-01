package storage

import (
	"context"
	"fmt"
	"maps"
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

	errorsMu   sync.RWMutex
	lastErrors = make(map[string]string)

	retiredMu sync.Mutex
	retired   []Storage

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
	maps.Copy(out, Storages)
	return out
}

// LastErrors returns the initialization errors from the last Reload.
func LastErrors() map[string]string {
	errorsMu.RLock()
	defer errorsMu.RUnlock()
	out := make(map[string]string, len(lastErrors))
	maps.Copy(out, lastErrors)
	return out
}

// ID returns the stable database ID for an initialized storage, or zero when
// the storage cannot be resolved from the current config.
func ID(st Storage) uint {
	if st == nil {
		return 0
	}
	if cfg := config.C().GetStorageByName(st.Name()); cfg != nil {
		return cfg.GetID()
	}
	return 0
}

// GetStorageByName returns storage by name from cache or creates a new one.
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

func GetStorageByID(ctx context.Context, id uint) (Storage, error) {
	cfg := config.C().GetStorageByID(id)
	if cfg == nil {
		return nil, fmt.Errorf("storage ID %d not found", id)
	}
	return GetStorageByName(ctx, cfg.GetName())
}

func GetStorageByUserIDAndID(ctx context.Context, chatID int64, id uint) (Storage, error) {
	if id == 0 {
		return nil, ErrStorageNameEmpty
	}
	if !config.C().HasStorageID(chatID, id) {
		return nil, fmt.Errorf("no storage ID %d for user %d", id, chatID)
	}
	return GetStorageByID(ctx, id)
}

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
	var storages []Storage
	for _, id := range config.C().GetStorageIDsByUserID(chatID) {
		storage, err := GetStorageByID(ctx, id)
		if err != nil {
			continue
		}
		storages = append(storages, storage)
	}
	return storages
}

// Reload rebuilds the storage registry from the current config snapshot.
func Reload(ctx context.Context) []error {
	logger := log.FromContext(ctx)
	logger.Debug("reloading storages...")

	initCtx := context.WithoutCancel(ctx)
	newStorages := make(map[string]Storage)
	errs := make(map[string]string)
	var out []error
	for _, storageCfg := range config.C().Storages {
		name := storageCfg.GetName()
		st, err := NewStorage(initCtx, storageCfg)
		if err != nil {
			errs[name] = err.Error()
			out = append(out, fmt.Errorf("storage %s: %w", name, err))
			logger.Errorf("failed to load storage %s: %v", name, err)
			continue
		}
		newStorages[name] = st
	}

	if err := ctx.Err(); err != nil {
		for _, st := range newStorages {
			if closer, ok := st.(StorageCloser); ok {
				closer.Close()
			}
		}
		return append(out, err)
	}

	storageMu.Lock()
	old := Storages
	Storages = newStorages
	storageMu.Unlock()

	// Keep retired instances alive: in-flight tasks may still hold them.
	retiredMu.Lock()
	for _, st := range old {
		retired = append(retired, st)
	}
	retiredMu.Unlock()

	errorsMu.Lock()
	lastErrors = errs
	errorsMu.Unlock()

	logger.Infof("successfully loaded %d storages", len(newStorages))
	return out
}

// CloseAll closes every initialized storage.
func CloseAll() {
	storageMu.Lock()
	all := make([]Storage, 0, len(Storages))
	for _, st := range Storages {
		all = append(all, st)
	}
	Storages = make(map[string]Storage)
	storageMu.Unlock()
	retiredMu.Lock()
	all = append(all, retired...)
	retired = nil
	retiredMu.Unlock()
	for _, st := range all {
		if closer, ok := st.(StorageCloser); ok {
			closer.Close()
		}
	}
}

// GetTelegramStorageByUserID returns the first enabled Telegram storage for the user.
func GetTelegramStorageByUserID(ctx context.Context, chatID int64) (Storage, error) {
	storages := GetUserStorages(ctx, chatID)
	for _, stor := range storages {
		if stor.Type() == storenum.Telegram {
			return stor, nil
		}
	}
	return nil, fmt.Errorf("no telegram storage found for user %d", chatID)
}
