package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/charmbracelet/log"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/rs/xid"
)

type S3 struct {
	config storconfig.S3StorageConfig
	client *s3.Client
	logger *log.Logger
}

func (m *S3) Init(ctx context.Context, cfg storconfig.StorageConfig) error {
	s3Config, ok := cfg.(*storconfig.S3StorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast s3 config")
	}
	if err := s3Config.Validate(); err != nil {
		return err
	}

	m.config = *s3Config
	m.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("s3[%s]", m.config.Name))
	loadOpts := make([]config.LoadOptionsFunc, 0)
	if m.config.Region != "" {
		loadOpts = append(loadOpts, config.WithRegion(m.config.Region))
	}
	if endpoint := m.config.Endpoint; endpoint != "" {
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			if m.config.UseSSL {
				endpoint = "https://" + endpoint
			} else {
				endpoint = "http://" + endpoint
			}
		}

		if _, err := url.Parse(endpoint); err != nil {
			return fmt.Errorf("invalid s3 endpoint %q: %w", m.config.Endpoint, err)
		}
		loadOpts = append(loadOpts, config.WithBaseEndpoint(endpoint))
	}
	loadOpts = append(loadOpts, config.WithCredentialsProvider(
		credentials.NewStaticCredentialsProvider(
			m.config.AccessKeyID,
			m.config.SecretAccessKey,
			"",
		),
	))
	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		func() []func(*config.LoadOptions) error {
			// wtf aws sdk
			// https://github.com/aws/aws-sdk-go-v2/issues/2193
			funcs := make([]func(*config.LoadOptions) error, 0)
			for _, fn := range loadOpts {
				funcs = append(funcs, fn)
			}
			return funcs
		}()...,
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	m.client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// Path style: https://s3.amazonaws.com/mybucket/path/to/file.jpg
		// virtual hosted style: https://mybucket.s3.amazonaws.com/path/to/file.jpg
		o.UsePathStyle = !m.config.VirtualHost
	})

	// Check if bucket exists
	_, err = m.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(m.config.BucketName),
	})
	if err != nil {
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

	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath

	// Unique filename
	for i := 1; m.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
		if i > 100 {
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

	// S3 PutObject needs either size or StreamingBody
	input := &s3.PutObjectInput{
		Bucket: aws.String(m.config.BucketName),
		Key:    aws.String(candidate),
		Body:   r,
	}

	if size >= 0 {
		input.ContentLength = &size
	}

	_, err := m.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (m *S3) Exists(ctx context.Context, storagePath string) bool {
	m.logger.Debugf("Checking if file exists at %s", storagePath)

	_, err := m.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(m.config.BucketName),
		Key:    aws.String(storagePath),
	})

	return err == nil
}
