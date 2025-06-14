package storage

import (
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
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

func (l *LocalStorageConfig) GetType() storenum.StorageType {
	return storenum.Local
}

func (l *LocalStorageConfig) GetName() string {
	return l.Name
}
