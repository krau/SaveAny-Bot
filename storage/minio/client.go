package minio

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/charmbracelet/log"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/xid"
)

type Minio struct {
	config config.MinioStorageConfig
	client *minio.Client
	logger *log.Logger
}

func (m *Minio) Init(ctx context.Context, cfg config.StorageConfig) error {
	minioConfig, ok := cfg.(*config.MinioStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast minio config")
	}
	if err := minioConfig.Validate(); err != nil {
		return err
	}
	m.config = *minioConfig
	m.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("minio[%s]", m.config.Name))

	client, err := minio.New(m.config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(m.config.AccessKeyID, m.config.SecretAccessKey, ""),
		Secure: m.config.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	exists, err := client.BucketExists(ctx, m.config.BucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %s does not exist", m.config.BucketName)
	}

	m.client = client
	return nil
}

func (m *Minio) Type() storenum.StorageType {
	return storenum.Minio
}

func (m *Minio) Name() string {
	return m.config.Name
}

func (m *Minio) JoinStoragePath(p string) string {
	return strings.TrimPrefix(path.Join(m.config.BasePath, p), "/")
}

func (m *Minio) Save(ctx context.Context, r io.Reader, storagePath string) error {
	m.logger.Infof("Saving file from reader to %s", storagePath)

	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath
	for i := 1; m.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
		if i > 1000 {
			m.logger.Errorf("Too many attempts to find a unique filename for %s", storagePath)
			candidate = fmt.Sprintf("%s_%s%s", base, xid.New().String(), ext)
			break
		}
	}
	size := int64(-1)
	if length := ctx.Value(ctxkey.ContentLength); length != nil {
		length, ok := length.(int64)
		if ok && length > 0 {
			size = length
		}
	}
	_, err := m.client.PutObject(ctx, m.config.BucketName, candidate, r, size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file to minio: %w", err)
	}

	return nil
}

func (m *Minio) Exists(ctx context.Context, storagePath string) bool {
	m.logger.Debugf("Checking if file exists at %s", storagePath)
	_, err := m.client.StatObject(ctx, m.config.BucketName, storagePath, minio.StatObjectOptions{})
	return err == nil
}
