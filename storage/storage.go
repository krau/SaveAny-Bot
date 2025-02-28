package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
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

type StreamStorage interface {
	Storage
	NewUploadStream(ctx context.Context, path string) (io.WriteCloser, error)
}

var Storages = make(map[string]Storage)

var UserStorages = make(map[int64][]Storage)

// GetStorageByName returns storage by name from cache or creates new one
func GetStorageByName(name string) (Storage, error) {
	if name == "" {
		return nil, ErrStorageNameEmpty
	}

	storage, ok := Storages[name]
	if ok {
		return storage, nil
	}
	cfg := config.Cfg.GetStorageByName(name)
	if cfg == nil {
		return nil, fmt.Errorf("未找到存储 %s", name)
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return nil, err
	}
	Storages[name] = storage
	return storage, nil
}

// 检查 user 是否可用指定的 storage, 若不可用则返回未找到错误
func GetStorageByUserIDAndName(chatID int64, name string) (Storage, error) {
	if name == "" {
		return nil, ErrStorageNameEmpty
	}

	if !config.Cfg.HasStorage(chatID, name) {
		return nil, fmt.Errorf("没有找到用户 %d 的存储 %s", chatID, name)
	}

	return GetStorageByName(name)
}

func GetUserStorages(chatID int64) []Storage {
	if chatID <= 0 {
		return nil
	}
	if storages, ok := UserStorages[chatID]; ok {
		return storages
	}
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
		return nil, fmt.Errorf("不支持的存储类型: %s", cfg.GetType())
	}

	storage := constructor()
	if err := storage.Init(cfg); err != nil {
		return nil, fmt.Errorf("初始化 %s 存储失败: %w", cfg.GetName(), err)
	}

	return storage, nil
}

func LoadStorages() {
	logger.L.Info("加载存储...")
	for _, storage := range config.Cfg.Storages {
		_, err := GetStorageByName(storage.GetName())
		if err != nil {
			logger.L.Errorf("加载存储 %s 失败: %v", storage.GetName(), err)
		}
	}
	logger.L.Infof("成功加载 %d 个存储", len(Storages))
	for user := range config.Cfg.GetUsersID() {
		UserStorages[int64(user)] = GetUserStorages(int64(user))
	}
}
