package storage

import (
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type TelegramStorageConfig struct {
	BaseConfig
	ChatID    int64 `toml:"chat_id" mapstructure:"chat_id" json:"chat_id"`
	ForceFile bool  `toml:"force_file" mapstructure:"force_file" json:"force_file"`
	RateLimit int   `toml:"rate_limit" mapstructure:"rate_limit" json:"rate_limit"`
	RateBurst int   `toml:"rate_burst" mapstructure:"rate_burst" json:"rate_burst"`
	SkipLarge bool  `toml:"skip_large" mapstructure:"skip_large" json:"skip_large"` // skip files larger than Telegram limit(2GB)
	// split files larger than Telegram limit(2GB) into parts of specified size, in MB, leave 0 to set default(2000MB)
	// only effective when SkipLarge is false
	// use zip when splitting
	SplitSizeMB int64 `toml:"split_size_mb" mapstructure:"split_size_mb" json:"split_size_mb"`
}

func (m *TelegramStorageConfig) Validate() error {
	if m.ChatID == 0 {
		return fmt.Errorf("chat_id is required for telegram storage")
	}
	if m.RateLimit < 0 || m.RateBurst < 0 {
		return fmt.Errorf("rate_limit and rate_burst must be greater than 0 for telegram storage")
	}
	return nil
}

func (m *TelegramStorageConfig) GetType() storenum.StorageType {
	return storenum.Telegram
}

func (m *TelegramStorageConfig) GetName() string {
	return m.Name
}
