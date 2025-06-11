package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func Init(ctx context.Context) {
	logger := log.FromContext(ctx)
	if err := os.MkdirAll(filepath.Dir(config.Cfg.DB.Path), 0755); err != nil {
		logger.Fatal("Failed to create data directory: ", err)
	}
	var err error
	db, err = gorm.Open(gormlite.Open(config.Cfg.DB.Path), &gorm.Config{
		Logger: glogger.New(logger, glogger.Config{
			Colorful:                  true,
			SlowThreshold:             time.Second * 5,
			LogLevel:                  glogger.Error,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
		PrepareStmt: true,
	})
	if err != nil {
		logger.Fatal("Failed to open database: ", err)
	}
	logger.Debug("Database connected")
	if err := db.AutoMigrate(&ReceivedFile{}, &User{}, &Dir{}, &CallbackData{}, &Rule{}); err != nil {
		logger.Fatal("迁移数据库失败, 如果您从旧版本升级, 建议手动删除数据库文件后重试: ", err)
	}
	if err := syncUsers(ctx); err != nil {
		logger.Fatal("Failed to sync users:", err)
	}
	logger.Debug("Database migrated")
	if config.Cfg.DB.Expire == 0 {
		return
	}
	if err := cleanExpiredData(db); err != nil {
		logger.Error("Failed to clean expired data: ", err)
	} else {
		logger.Debug("Cleaned expired data")
	}
	go cleanJob(ctx, db)
	logger.Debug("Database initialized")
}

func syncUsers(ctx context.Context) error {
	logger := log.FromContext(ctx)
	dbUsers, err := GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	dbUserMap := make(map[int64]User)
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
			logger.Infof("创建用户: %d", cfgID)
		}
	}

	for dbID, dbUser := range dbUserMap {
		if _, exists := cfgUserMap[dbID]; !exists {
			if err := DeleteUser(&dbUser); err != nil {
				return fmt.Errorf("failed to delete user %d: %w", dbID, err)
			}
			logger.Infof("删除用户: %d", dbID)
		}
	}

	return nil
}

func cleanExpiredData(db *gorm.DB) error {
	var fileErr error
	if err := db.Where("updated_at < ?", time.Now().Add(-time.Duration(config.Cfg.DB.Expire)*time.Second)).Unscoped().Delete(&ReceivedFile{}).Error; err != nil {
		fileErr = fmt.Errorf("failed to delete expired files: %w", err)
	}
	var cbErr error
	if err := db.Where("updated_at < ?", time.Now().Add(-time.Duration(config.Cfg.DB.Expire)*time.Second)).Unscoped().Delete(&CallbackData{}).Error; err != nil {
		cbErr = fmt.Errorf("failed to delete expired callback data: %w", err)
	}
	return errors.Join(fileErr, cbErr)
}

func cleanJob(ctx context.Context, db *gorm.DB) {
	tick := time.NewTicker(time.Duration(config.Cfg.DB.Expire) * time.Second)
	defer tick.Stop()
	for range tick.C {
		if err := cleanExpiredData(db); err != nil {
			log.FromContext(ctx).Error("Failed to clean expired data: ", err)
		} else {
			log.FromContext(ctx).Debug("Cleaned expired data")
		}
	}
}
