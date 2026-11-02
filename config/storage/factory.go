package storage

import (
	"encoding/json"
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/spf13/viper"
)

// NewStorageConfigFromJSON builds a typed storage config from JSON-encoded
// type-specific fields. Only the legacy importer still needs Viper; runtime
// storage settings come from the database as JSON.
func NewStorageConfigFromJSON(id uint, name, typ string, enable bool, raw []byte) (StorageConfig, error) {
	st, err := storenum.ParseStorageType(typ)
	if err != nil {
		return nil, fmt.Errorf("invalid storage type %s: %w", typ, err)
	}

	base := BaseConfig{ID: id, Name: name, Type: typ, Enable: enable}
	var cfg StorageConfig
	switch st {
	case storenum.Local:
		cfg = &LocalStorageConfig{BaseConfig: base}
	case storenum.Alist:
		cfg = &AlistStorageConfig{BaseConfig: base}
	case storenum.Webdav:
		cfg = &WebdavStorageConfig{BaseConfig: base}
	case storenum.S3:
		cfg = &S3StorageConfig{BaseConfig: base}
	case storenum.Telegram:
		cfg = &TelegramStorageConfig{BaseConfig: base}
	case storenum.Rclone:
		cfg = &RcloneStorageConfig{BaseConfig: base}
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", typ)
	}

	if len(raw) > 0 {
		if err := json.Unmarshal(raw, cfg); err != nil {
			return nil, fmt.Errorf("failed to decode %s storage config: %w", typ, err)
		}
	}
	// The database owns identity and enablement; type-specific JSON must not
	// override them.
	setBaseConfig(cfg, base)
	return cfg, nil
}

func setBaseConfig(cfg StorageConfig, base BaseConfig) {
	switch typed := cfg.(type) {
	case *LocalStorageConfig:
		typed.BaseConfig = base
	case *AlistStorageConfig:
		typed.BaseConfig = base
	case *WebdavStorageConfig:
		typed.BaseConfig = base
	case *S3StorageConfig:
		typed.BaseConfig = base
	case *TelegramStorageConfig:
		typed.BaseConfig = base
	case *RcloneStorageConfig:
		typed.BaseConfig = base
	}
}

// LoadAllBaseConfigs unmarshals every legacy storage entry, including disabled
// ones. It is only used by the one-time legacy importer.
func LoadAllBaseConfigs(v *viper.Viper) ([]BaseConfig, error) {
	var baseConfigs []BaseConfig
	if err := v.UnmarshalKey("storages", &baseConfigs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage configs: %w", err)
	}
	return baseConfigs, nil
}
