package storage

import (
	"encoding/json"
	"fmt"
	"reflect"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var storageConfigPrototypes = map[storenum.StorageType]StorageConfig{
	storenum.Local:    &LocalStorageConfig{},
	storenum.Alist:    &AlistStorageConfig{},
	storenum.Webdav:   &WebdavStorageConfig{},
	storenum.Minio:    &MinioStorageConfig{},
	storenum.S3:       &S3StorageConfig{},
	storenum.Telegram: &TelegramStorageConfig{},
	storenum.Rclone:   &RcloneStorageConfig{},
}

var storageFactories = map[storenum.StorageType]func(cfg *BaseConfig) (StorageConfig, error){
	storenum.Local:    createStorageConfig(&LocalStorageConfig{}),
	storenum.Alist:    createStorageConfig(&AlistStorageConfig{}),
	storenum.Webdav:   createStorageConfig(&WebdavStorageConfig{}),
	storenum.Minio:    createStorageConfig(&MinioStorageConfig{}),
	storenum.S3:       createStorageConfig(&S3StorageConfig{}),
	storenum.Telegram: createStorageConfig(&TelegramStorageConfig{}),
	storenum.Rclone:   createStorageConfig(&RcloneStorageConfig{}),
}

func createStorageConfig(configType StorageConfig) func(cfg *BaseConfig) (StorageConfig, error) {
	return func(cfg *BaseConfig) (StorageConfig, error) {
		configValue := reflect.New(reflect.TypeOf(configType).Elem()).Interface().(StorageConfig)

		reflect.ValueOf(configValue).Elem().FieldByName("BaseConfig").Set(reflect.ValueOf(*cfg))

		if err := mapstructure.Decode(cfg.RawConfig, configValue); err != nil {
			return nil, fmt.Errorf("failed to decode %s storage config: %w", cfg.Type, err)
		}

		return configValue, nil
	}
}

func LoadStorageConfigs(v *viper.Viper) ([]StorageConfig, error) {
	var baseConfigs []BaseConfig
	if err := v.UnmarshalKey("storages", &baseConfigs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage configs: %w", err)
	}

	var configs []StorageConfig
	for _, baseCfg := range baseConfigs {
		if !baseCfg.Enable {
			continue
		}
		st, err := storenum.ParseStorageType(baseCfg.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid storage type %s for %s: %w", baseCfg.Type, baseCfg.Name, err)
		}

		factory, ok := storageFactories[st]
		if !ok {
			return nil, fmt.Errorf("unsupported storage type: %s", baseCfg.Type)
		}

		cfg, err := factory(&baseCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage config for %s: %w", baseCfg.Name, err)
		}

		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid storage config for %s: %w", baseCfg.Name, err)
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

// NewStorageConfig builds a storage config from a BaseConfig carrying remaining keys.
func NewStorageConfig(base *BaseConfig) (StorageConfig, error) {
	st, err := storenum.ParseStorageType(base.Type)
	if err != nil {
		return nil, err
	}
	factory, ok := storageFactories[st]
	if !ok {
		return nil, fmt.Errorf("unsupported storage type: %s", base.Type)
	}
	return factory(base)
}

// NewStorageConfigFromJSON builds a storage config from JSON-encoded type-specific fields.
func NewStorageConfigFromJSON(name, typ string, enable bool, raw []byte) (StorageConfig, error) {
	st, err := storenum.ParseStorageType(typ)
	if err != nil {
		return nil, fmt.Errorf("invalid storage type %s: %w", typ, err)
	}
	proto, ok := storageConfigPrototypes[st]
	if !ok {
		return nil, fmt.Errorf("unsupported storage type: %s", typ)
	}
	configValue := reflect.New(reflect.TypeOf(proto).Elem()).Interface().(StorageConfig)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, configValue); err != nil {
			return nil, fmt.Errorf("failed to decode %s storage config: %w", typ, err)
		}
	}
	reflect.ValueOf(configValue).Elem().FieldByName("BaseConfig").Set(reflect.ValueOf(BaseConfig{
		Name: name, Type: typ, Enable: enable,
	}))
	return configValue, nil
}

// LoadAllBaseConfigs unmarshals every storage entry, including disabled ones.
func LoadAllBaseConfigs(v *viper.Viper) ([]BaseConfig, error) {
	var baseConfigs []BaseConfig
	if err := v.UnmarshalKey("storages", &baseConfigs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage configs: %w", err)
	}
	return baseConfigs, nil
}
