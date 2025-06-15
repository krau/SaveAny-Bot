package storage

import (
	"fmt"
	"reflect"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var storageFactories = map[storenum.StorageType]func(cfg *BaseConfig) (StorageConfig, error){
	storenum.Local:    createStorageConfig(&LocalStorageConfig{}),
	storenum.Alist:    createStorageConfig(&AlistStorageConfig{}),
	storenum.Webdav:   createStorageConfig(&WebdavStorageConfig{}),
	storenum.Minio:    createStorageConfig(&MinioStorageConfig{}),
	storenum.Telegram: createStorageConfig(&TelegramStorageConfig{}),
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
