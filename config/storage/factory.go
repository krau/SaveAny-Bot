package storage

import (
	"fmt"
	"reflect"

	"github.com/krau/SaveAny-Bot/types"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var storageFactories = map[types.StorageType]func(cfg *BaseConfig) (StorageConfig, error){
	types.StorageTypeLocal:  createStorageConfig(&LocalStorageConfig{}),
	types.StorageTypeAlist:  createStorageConfig(&AlistStorageConfig{}),
	types.StorageTypeWebdav: createStorageConfig(&WebdavStorageConfig{}),
	types.StorageTypeMinio:  createStorageConfig(&MinioStorageConfig{}),
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

		factory, ok := storageFactories[types.StorageType(baseCfg.Type)]
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
