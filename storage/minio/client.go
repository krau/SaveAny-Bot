package minio

import (
	"context"
	"fmt"
	"path"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Minio struct {
	config config.MinioStorageConfig
	client *minio.Client
}

func (m *Minio) Init(cfg config.StorageConfig) error {
	minioConfig, ok := cfg.(*config.MinioStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast minio config")
	}
	if err := minioConfig.Validate(); err != nil {
		return err
	}
	m.config = *minioConfig

	client, err := minio.New(m.config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(m.config.AccessKeyID, m.config.SecretAccessKey, ""),
		Secure: m.config.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	exists, err := client.BucketExists(context.Background(), m.config.BucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %s does not exist", m.config.BucketName)
	}

	m.client = client
	return nil
}

func (m *Minio) Type() types.StorageType {
	return types.StorageTypeMinio
}

func (m *Minio) Name() string {
	return m.config.Name
}

func (m *Minio) JoinStoragePath(task types.Task) string {
	return path.Join(m.config.BasePath, task.StoragePath)
}

func (m *Minio) Save(ctx context.Context, localFilePath, storagePath string) error {
	logger.L.Infof("Saving file %s to %s", localFilePath, storagePath)

	_, err := m.client.FPutObject(ctx, m.config.BucketName, storagePath, localFilePath, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file to minio: %w", err)
	}

	return nil
}
