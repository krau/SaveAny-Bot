package storage

import (
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type S3StorageConfig struct {
	BaseConfig
	Endpoint        string `toml:"endpoint" mapstructure:"endpoint" json:"endpoint"`
	AccessKeyID     string `toml:"access_key_id" mapstructure:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key" mapstructure:"secret_access_key" json:"secret_access_key"`
	BucketName      string `toml:"bucket_name" mapstructure:"bucket_name" json:"bucket_name"`
	UseSSL          bool   `toml:"use_ssl" mapstructure:"use_ssl" json:"use_ssl"`
	BasePath        string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
	Region          string `toml:"region" mapstructure:"region" json:"region"`
	VirtualHost     bool   `toml:"virtual_host" mapstructure:"virtual_host" json:"virtual_host"`
}

func (m *S3StorageConfig) Validate() error {
	if m.Endpoint == "" {
		return fmt.Errorf("endpoint is required for s3 storage")
	}
	if m.AccessKeyID == "" || m.SecretAccessKey == "" {
		return fmt.Errorf("access_key_id and secret_access_key are required for s3 storage")
	}
	if m.BucketName == "" {
		return fmt.Errorf("bucket_name is required for s3 storage")
	}
	if m.BasePath == "" {
		return fmt.Errorf("base_path is required for s3 storage")
	}
	return nil
}

func (m *S3StorageConfig) GetType() storenum.StorageType {
	return storenum.S3
}

func (m *S3StorageConfig) GetName() string {
	return m.Name
}
