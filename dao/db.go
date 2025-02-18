package dao

import (
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/types"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func Init() {
	if err := os.MkdirAll(filepath.Dir(config.Cfg.DB.Path), 0755); err != nil {
		logger.L.Fatal("Failed to create data directory: ", err)
		os.Exit(1)
	}
	var err error
	db, err = gorm.Open(sqlite.Open(config.Cfg.DB.Path), &gorm.Config{
		Logger: glogger.New(logger.L, glogger.Config{
			Colorful:                  true,
			SlowThreshold:             time.Second * 5,
			LogLevel:                  glogger.Error,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
		PrepareStmt: true,
	})
	if err != nil {
		logger.L.Fatal("Failed to open database: ", err)
		os.Exit(1)
	}
	logger.L.Debug("Database connected")
	if err := db.AutoMigrate(&types.ReceivedFile{}, &types.User{}, &types.StorageModel{}); err != nil {
		logger.L.Fatal("迁移数据库失败, 如果您从旧版本升级, 建议手动删除数据库文件后重试: ", err)
	}

	for _, admin := range config.Cfg.Telegram.Admins {
		CreateUser(int64(admin))
	}

	logger.L.Infof("Migrating config storages to users")
	storageCfg := config.Cfg.Storage

	allUsers, err := GetAllUsers()
	if err != nil {
		logger.L.Fatalf("Failed to get all users: %v", err)
	} else {
		for _, user := range allUsers {
			found := false
			for _, admin := range config.Cfg.Telegram.Admins {
				if user.ChatID == int64(admin) {
					found = true
					break
				}
			}
			if !found {
				logger.L.Debugf("Deleting user %d", user.ChatID)
				if err := DeleteUser(&user); err != nil {
					logger.L.Fatalf("Failed to delete user %d: %v", user.ChatID, err)
				}
			}
		}
	}
	// TODO: refactor this
	for _, admin := range config.Cfg.Telegram.Admins {
		user, err := GetUserByChatID(int64(admin))
		if err != nil {
			logger.L.Fatalf("Failed to get user by chat ID %d: %v", admin, err)
			continue
		}
		if len(user.Storages) > 0 {
			logger.L.Debugf("User %d already has storages", admin)
			continue
		}
		if storageCfg.Alist.Enable {
			alistStorage := &types.StorageModel{
				Type:   string(types.StorageTypeAlist),
				Active: true,
				Config: storageCfg.Alist.ToJSON(),
			}
			hash := alistStorage.GenHash()
			alistStorage.Hash = hash
			if storagedb, err := GetStorageByHash(hash); err == nil {
				logger.L.Debugf("Alist storage already exists")
				user.Storages = append(user.Storages, storagedb)
			} else {
				id, err := CreateStorage(alistStorage)
				if err != nil {
					logger.L.Fatalf("Failed to create storage: %v", err)
				} else {
					storagedb := &types.StorageModel{}
					storagedb.ID = id
					user.Storages = append(user.Storages, storagedb)
				}
			}
		}
		if storageCfg.Local.Enable {
			localStorage := &types.StorageModel{
				Type:   string(types.StorageTypeLocal),
				Active: true,
				Config: storageCfg.Local.ToJSON(),
			}
			hash := localStorage.GenHash()
			localStorage.Hash = hash
			if storagedb, err := GetStorageByHash(hash); err == nil {
				logger.L.Debugf("Local storage already exists")
				user.Storages = append(user.Storages, storagedb)
			} else {
				id, err := CreateStorage(localStorage)
				if err != nil {
					logger.L.Fatalf("Failed to create storage: %v", err)
				} else {
					storagedb := &types.StorageModel{}
					storagedb.ID = id
					user.Storages = append(user.Storages, storagedb)
				}
			}
		}
		if storageCfg.Webdav.Enable {
			webdavStorage := &types.StorageModel{
				Type:   string(types.StorageTypeWebdav),
				Active: true,
				Config: storageCfg.Webdav.ToJSON(),
			}
			hash := webdavStorage.GenHash()
			webdavStorage.Hash = hash
			if storagedb, err := GetStorageByHash(hash); err == nil {
				logger.L.Debugf("Webdav storage already exists")
				user.Storages = append(user.Storages, storagedb)
			} else {
				id, err := CreateStorage(webdavStorage)
				if err != nil {
					logger.L.Fatalf("Failed to create storage: %v", err)
				} else {
					storagedb := &types.StorageModel{}
					storagedb.ID = id
					user.Storages = append(user.Storages, storagedb)
				}
			}
		}
		if err := UpdateUser(user); err != nil {
			logger.L.Fatalf("Failed to update user with storages: %v", err)
		}
	}
	logger.L.Infof("Migration done")
}
