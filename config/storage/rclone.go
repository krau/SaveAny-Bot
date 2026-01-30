package storage

import (
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type RcloneStorageConfig struct {
	BaseConfig
	// The name of the remote as defined in rclone config
	Remote   string `toml:"remote" mapstructure:"remote" json:"remote"`
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
	// The path to the rclone config file, if not using the default
	ConfigPath string `toml:"config_path" mapstructure:"config_path" json:"config_path"`
	// Additional flags to pass to rclone commands
	Flags []string `toml:"flags" mapstructure:"flags" json:"flags"`
}

func (r *RcloneStorageConfig) Validate() error {
	if r.Remote == "" {
		return fmt.Errorf("remote is required for rclone storage")
	}
	return nil
}

func (r *RcloneStorageConfig) GetType() storenum.StorageType {
	return storenum.Rclone
}

func (r *RcloneStorageConfig) GetName() string {
	return r.Name
}
