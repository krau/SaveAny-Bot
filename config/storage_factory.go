package config

import (
	"fmt"

	"github.com/krau/SaveAny-Bot/types"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type StorageConfig interface {
	Validate() error
	GetType() types.StorageType
	GetName() string
}

// Base storage config
type NewStorageConfig struct {
	Name      string         `toml:"name" mapstructure:"name" json:"name"`
	Type      string         `toml:"type" mapstructure:"type" json:"type"`
	Enable    bool           `toml:"enable" mapstructure:"enable" json:"enable"`
	RawConfig map[string]any `toml:"-" mapstructure:",remain"`
}

var storageFactories = map[types.StorageType]func(cfg *NewStorageConfig) (StorageConfig, error){
	types.StorageTypeLocal:  newLocalStorageConfig,
	types.StorageTypeAlist:  newAlistStorageConfig,
	types.StorageTypeWebdav: newWebdavStorageConfig,
	types.StorageTypeMinio:  newMinioStorageConfig,
}

func LoadStorageConfigs(v *viper.Viper) ([]StorageConfig, error) {
	var baseConfigs []NewStorageConfig
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

func newLocalStorageConfig(cfg *NewStorageConfig) (StorageConfig, error) {
	var localCfg LocalStorageConfig
	localCfg.NewStorageConfig = *cfg

	if err := mapstructure.Decode(cfg.RawConfig, &localCfg); err != nil {
		return nil, fmt.Errorf("failed to decode local storage config: %w", err)
	}

	return &localCfg, nil
}

func newAlistStorageConfig(cfg *NewStorageConfig) (StorageConfig, error) {
	var alistCfg AlistStorageConfig
	alistCfg.NewStorageConfig = *cfg

	if err := mapstructure.Decode(cfg.RawConfig, &alistCfg); err != nil {
		return nil, fmt.Errorf("failed to decode alist storage config: %w", err)
	}

	return &alistCfg, nil
}

func newWebdavStorageConfig(cfg *NewStorageConfig) (StorageConfig, error) {
	var webdavCfg WebdavStorageConfig
	webdavCfg.NewStorageConfig = *cfg

	if err := mapstructure.Decode(cfg.RawConfig, &webdavCfg); err != nil {
		return nil, fmt.Errorf("failed to decode webdav storage config: %w", err)
	}

	return &webdavCfg, nil
}

func newMinioStorageConfig(cfg *NewStorageConfig) (StorageConfig, error) {
	var minioCfg MinioStorageConfig
	minioCfg.NewStorageConfig = *cfg
	if err := mapstructure.Decode(cfg.RawConfig, &minioCfg); err != nil {
		return nil, fmt.Errorf("failed to decode minio storage config: %w", err)
	}
	return &minioCfg, nil
}
