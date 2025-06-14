package storage

import (
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type AlistStorageConfig struct {
	BaseConfig
	URL      string `toml:"url" mapstructure:"url" json:"url"`
	Username string `toml:"username" mapstructure:"username" json:"username"`
	Password string `toml:"password" mapstructure:"password" json:"password"`
	Token    string `toml:"token" mapstructure:"token" json:"token"`
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
	TokenExp int64  `toml:"token_exp" mapstructure:"token_exp" json:"token_exp"`
}

func (a *AlistStorageConfig) Validate() error {
	if a.URL == "" {
		return fmt.Errorf("url is required for alist storage")
	}
	if a.Token == "" && (a.Username == "" || a.Password == "") {
		return fmt.Errorf("username and password or token is required for alist storage")
	}
	if a.BasePath == "" {
		return fmt.Errorf("base_path is required for alist storage")
	}
	return nil
}

func (a *AlistStorageConfig) GetType() storenum.StorageType {
	return storenum.Alist
}

func (a *AlistStorageConfig) GetName() string {
	return a.Name
}
