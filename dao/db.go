package dao

import (
	"fmt"
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
	if err := db.AutoMigrate(&types.ReceivedFile{}, &types.User{}); err != nil {
		logger.L.Fatal("迁移数据库失败, 如果您从旧版本升级, 建议手动删除数据库文件后重试: ", err)
	}

	if err := syncUsers(); err != nil {
		logger.L.Fatal("Failed to sync users:", err)
	}
}

func syncUsers() error {
	dbUsers, err := GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	dbUserMap := make(map[int64]types.User)
	for _, u := range dbUsers {
		dbUserMap[u.ChatID] = u
	}

	cfgUserMap := make(map[int64]struct{})
	for _, u := range config.Cfg.Users {
		cfgUserMap[u.ID] = struct{}{}
	}

	for cfgID := range cfgUserMap {
		if _, exists := dbUserMap[cfgID]; !exists {
			if err := CreateUser(cfgID); err != nil {
				return fmt.Errorf("failed to create user %d: %w", cfgID, err)
			}
			logger.L.Infof("创建用户: %d", cfgID)
		}
	}

	for dbID, dbUser := range dbUserMap {
		if _, exists := cfgUserMap[dbID]; !exists {
			if err := DeleteUser(&dbUser); err != nil {
				return fmt.Errorf("failed to delete user %d: %w", dbID, err)
			}
			logger.L.Infof("删除用户: %d", dbID)
		}
	}

	return nil
}
