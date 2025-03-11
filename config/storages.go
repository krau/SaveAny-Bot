package config

import (
	"fmt"

	"github.com/krau/SaveAny-Bot/types"
)

func (c *Config) GetStoragesByType(storageType types.StorageType) []StorageConfig {
	var storages []StorageConfig
	for _, storage := range c.Storages {
		if storage.GetType() == storageType {
			storages = append(storages, storage)
		}
	}
	return storages
}

func (c *Config) GetStorageByName(name string) StorageConfig {
	for _, storage := range c.Storages {
		if storage.GetName() == name {
			return storage
		}
	}
	return nil
}

type LocalStorageConfig struct {
	NewStorageConfig
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

type AlistStorageConfig struct {
	NewStorageConfig
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

func (a *AlistStorageConfig) GetType() types.StorageType {
	return types.StorageTypeAlist
}

func (a *AlistStorageConfig) GetName() string {
	return a.Name
}

type WebdavStorageConfig struct {
	NewStorageConfig
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

type MinioStorageConfig struct {
	NewStorageConfig
	Endpoint        string `toml:"endpoint" mapstructure:"endpoint" json:"endpoint"`
	AccessKeyID     string `toml:"access_key_id" mapstructure:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key" mapstructure:"secret_access_key" json:"secret_access_key"`
	BucketName      string `toml:"bucket_name" mapstructure:"bucket_name" json:"bucket_name"`
	UseSSL          bool   `toml:"use_ssl" mapstructure:"use_ssl" json:"use_ssl"`
	BasePath        string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
}

func (m *MinioStorageConfig) Validate() error {
	if m.Endpoint == "" {
		return fmt.Errorf("endpoint is required for minio storage")
	}
	if m.AccessKeyID == "" || m.SecretAccessKey == "" {
		return fmt.Errorf("access_key_id and secret_access_key are required for minio storage")
	}
	if m.BucketName == "" {
		return fmt.Errorf("bucket_name is required for minio storage")
	}
	if m.BasePath == "" {
		return fmt.Errorf("base_path is required for minio storage")
	}
	return nil
}

func (m *MinioStorageConfig) GetType() types.StorageType {
	return types.StorageTypeMinio
}

func (m *MinioStorageConfig) GetName() string {
	return m.Name
}
