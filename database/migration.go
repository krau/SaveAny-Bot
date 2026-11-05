package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CurrentSchemaVersion is the database schema expected by this build.
// Bump it together with a new entry in schemaMigrations.
const CurrentSchemaVersion = 1

type schemaMigration struct {
	version int
	up      func(tx *gorm.DB) error
}

var schemaMigrations = []schemaMigration{
	{version: 1, up: baselineSchema},
}

func baselineSchema(tx *gorm.DB) error {
	return tx.AutoMigrate(
		&User{},
		&Dir{},
		&Rule{},
		&WatchChat{},
		&Setting{},
		&StorageRecord{},
		&TaskRecord{},
		&TaskItemRecord{},
		&WebSession{},
		&BotWorker{},
	)
}

func migrate(ctx context.Context) error {
	if err := db.WithContext(ctx).Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	applied_at DATETIME NOT NULL
)`).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var applied int
	if err := db.WithContext(ctx).Raw("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&applied).Error; err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if applied > CurrentSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", applied, CurrentSchemaVersion)
	}
	for _, migration := range schemaMigrations {
		if migration.version <= applied {
			continue
		}
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := migration.up(tx); err != nil {
				return err
			}
			return tx.Exec(
				"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
				migration.version, time.Now().UTC(),
			).Error
		}); err != nil {
			return fmt.Errorf("apply schema migration %d: %w", migration.version, err)
		}
	}
	return nil
}
