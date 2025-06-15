package storage

import (
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type StorageConfig interface {
	Validate() error
	GetType() storenum.StorageType
	GetName() string
}

type BaseConfig struct {
	Name      string         `toml:"name" mapstructure:"name" json:"name"`
	Type      string         `toml:"type" mapstructure:"type" json:"type"`
	Enable    bool           `toml:"enable" mapstructure:"enable" json:"enable"`
	RawConfig map[string]any `toml:"-" mapstructure:",remain"`
}
