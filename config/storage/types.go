package storage

import (
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type StorageConfig interface {
	Validate() error
	GetID() uint
	GetType() storenum.StorageType
	GetName() string
}

type BaseConfig struct {
	ID        uint           `toml:"-" mapstructure:"-" json:"-"`
	Name      string         `toml:"name" mapstructure:"name" json:"name"`
	Type      string         `toml:"type" mapstructure:"type" json:"type"`
	Enable    bool           `toml:"enable" mapstructure:"enable" json:"enable"`
	RawConfig map[string]any `toml:"-" mapstructure:",remain"`
}

func (b BaseConfig) GetID() uint { return b.ID }
