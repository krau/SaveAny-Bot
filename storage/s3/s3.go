package s3

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/charmbracelet/log"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/s3"
	"github.com/rs/xid"
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

	// Check if bucket exists
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

func (m *S3) JoinStoragePath(p string) string {
	return strings.TrimPrefix(path.Join(m.config.BasePath, p), "/")
}

func (m *S3) Save(ctx context.Context, r io.Reader, storagePath string) error {
	m.logger.Infof("Saving file from reader to %s", storagePath)
	storagePath = m.JoinStoragePath(storagePath)
	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath

	// Unique filename
	for i := 1; m.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
		if i > 10 {
			m.logger.Errorf("Too many attempts for unique filename: %s", storagePath)
			candidate = fmt.Sprintf("%s_%s%s", base, xid.New().String(), ext)
			break
		}
	}

	// Determine content length
	size := int64(-1)
	if length := ctx.Value(ctxkey.ContentLength); length != nil {
		if l, ok := length.(int64); ok && l > 0 {
			size = l
		}
	}

	err := m.client.Put(ctx, candidate, r, size)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (m *S3) Exists(ctx context.Context, storagePath string) bool {
	m.logger.Debugf("Checking if file exists at %s", storagePath)

	return m.client.Exists(ctx, storagePath)
}
