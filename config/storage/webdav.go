package storage

import (
	"fmt"

	"github.com/krau/SaveAny-Bot/types"
)

type WebdavStorageConfig struct {
	BaseConfig
	URL      string `toml:"url" mapstructure:"url" json:"url"`
	Username string `toml:"username" mapstructure:"username" json:"username"`
	Password string `toml:"password" mapstructure:"password" json:"password"`
	BasePath string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
}

func (w *WebdavStorageConfig) Validate() error {
	if w.URL == "" {
		return fmt.Errorf("url is required for webdav storage")
	}
	if w.Username == "" || w.Password == "" {
		return fmt.Errorf("username and password is required for webdav storage")
	}
	if w.BasePath == "" {
		return fmt.Errorf("base_path is required for webdav storage")
	}
	return nil
}

func (w *WebdavStorageConfig) GetType() types.StorageType {
	return types.StorageTypeWebdav
}

func (w *WebdavStorageConfig) GetName() string {
	return w.Name
}
