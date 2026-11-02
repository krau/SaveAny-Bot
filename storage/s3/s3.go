package s3

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/s3"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
)

type S3 struct {
	config storconfig.S3StorageConfig
	client *s3.Client
	logger *log.Logger
}

func (m *S3) Init(ctx context.Context, cfg storconfig.StorageConfig) error {
	s3cfg, ok := cfg.(*storconfig.S3StorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast s3 config")
	}
	if err := s3cfg.Validate(); err != nil {
		return err
	}
	m.config = *s3cfg
	m.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("s3[%s]", m.config.Name))
	client, err := s3.NewClient(&s3.Config{
		Endpoint:        m.config.Endpoint,
		Region:          m.config.Region,
		AccessKeyID:     m.config.AccessKeyID,
		SecretAccessKey: m.config.SecretAccessKey,
		BucketName:      m.config.BucketName,
		PathStyle:       !m.config.VirtualHost,
	})
	if err != nil {
		return fmt.Errorf("failed to create s3 client: %w", err)
	}
	m.client = client

	if err := m.client.HeadBucket(ctx); err != nil {
		return fmt.Errorf("bucket %s not accessible: %w", m.config.BucketName, err)
	}
	return nil
}

func (m *S3) Type() storenum.StorageType {
	return storenum.S3
}

func (m *S3) Name() string {
	return m.config.Name
}

func (m *S3) JoinStoragePath(p string) (string, error) {
	return storagetypes.SecureJoin(m.config.BasePath, p)
}

func (m *S3) Save(ctx context.Context, r io.Reader, storagePath string) error {
	m.logger.Infof("Saving file from reader to %s", storagePath)
	candidate, err := m.JoinStoragePath(storagePath)
	if err != nil {
		return err
	}

	overwrite, _ := ctx.Value(ctxkey.OverwriteExisting).(bool)
	if !overwrite {
		candidate, err = fsutil.UniquePath(strings.TrimPrefix(m.config.BasePath, "/"), storagePath, func(c string) (bool, error) {
			return m.existsKey(ctx, c)
		}, 10)
		if err != nil {
			return err
		}
	}

	size := int64(-1)
	if length := ctx.Value(ctxkey.ContentLength); length != nil {
		if l, ok := length.(int64); ok && l > 0 {
			size = l
		}
	}

	if overwrite {
		err = m.client.Put(ctx, candidate, r, size)
	} else {
		err = m.client.PutIfAbsent(ctx, candidate, r, size)
	}
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (m *S3) Exists(ctx context.Context, storagePath string) (bool, error) {
	m.logger.Debugf("Checking if file exists at %s", storagePath)

	key, err := m.JoinStoragePath(storagePath)
	if err != nil {
		return false, err
	}
	return m.existsKey(ctx, key)
}

func (m *S3) existsKey(ctx context.Context, key string) (bool, error) {
	return m.client.Exists(ctx, key)
}
