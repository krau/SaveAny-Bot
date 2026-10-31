package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func Init(ctx context.Context, path string) {
	logger := log.FromContext(ctx)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logger.Fatalf("Failed to create data directory: %v", err)
	}
	var err error
	db, err = gorm.Open(GetDialect(path), &gorm.Config{
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
		logger.Fatalf("Failed to open database: %v", err)
	}
	_ = os.Chmod(path, 0o600)
	logger.Debug("Database connected")
	if err := db.AutoMigrate(&User{}, &Dir{}, &Rule{}, &WatchChat{}, &Setting{}, &StorageRecord{}, &TaskRecord{}, &WebSession{}); err != nil {
		logger.Fatalf("Database migration failed: %v", err)
	}
	if err := MigrateLegacyPeerIDs(ctx); err != nil {
		logger.Errorf("Failed to migrate legacy peer IDs: %v", err)
	}
	logger.Info("Database initialized")
}

func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql db: %w", err)
	}
	return sqlDB.Close()
}

func Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
