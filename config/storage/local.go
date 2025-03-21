package storage

import (
	"fmt"

	"github.com/krau/SaveAny-Bot/types"
)

type LocalStorageConfig struct {
	BaseConfig
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
}

func (l *LocalStorageConfig) Validate() error {
	if l.BasePath == "" {
		return fmt.Errorf("path is required for local storage")
	}
	return nil
}

func (l *LocalStorageConfig) GetType() types.StorageType {
	return types.StorageTypeLocal
}

func (l *LocalStorageConfig) GetName() string {
	return l.Name
}
