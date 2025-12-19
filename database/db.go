package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func Init(ctx context.Context) {
	logger := log.FromContext(ctx)
	if err := os.MkdirAll(filepath.Dir(config.C().DB.Path), 0755); err != nil {
		logger.Fatal("Failed to create data directory: ", err)
	}
	var err error
	db, err = gorm.Open(GetDialect(config.C().DB.Path), &gorm.Config{
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
	if err := db.AutoMigrate(&User{}, &Dir{}, &Rule{}, &WatchChat{}); err != nil {
		logger.Fatal("Database migration failed; if upgrading from an old version, try deleting the database file and retrying", "error", err)
	}
	if err := syncUsers(ctx); err != nil {
		logger.Fatal("Failed to sync users:", err)
	}
	logger.Debug("Database migrated")
	logger.Info("Database initialized")
}

func syncUsers(ctx context.Context) error {
	logger := log.FromContext(ctx)
	dbUsers, err := GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	dbUserMap := make(map[int64]User)
	for _, u := range dbUsers {
		dbUserMap[u.ChatID] = u
	}

	cfgUserMap := make(map[int64]struct{})
	for _, u := range config.C().Users {
		cfgUserMap[u.ID] = struct{}{}
	}

	for cfgID := range cfgUserMap {
		if _, exists := dbUserMap[cfgID]; !exists {
			if err := CreateUser(ctx, cfgID); err != nil {
				return fmt.Errorf("failed to create user %d: %w", cfgID, err)
			}
			logger.Infof("Created user from config: %d", cfgID)
		}
	}

	for dbID, dbUser := range dbUserMap {
		if _, exists := cfgUserMap[dbID]; !exists {
			if err := DeleteUser(ctx, &dbUser); err != nil {
				return fmt.Errorf("failed to delete user %d: %w", dbID, err)
			}
			logger.Infof("Deleted user not present in config: %d", dbID)
		}
	}

	return nil
}
