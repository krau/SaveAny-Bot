//go:build no_minio

package minio

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	config "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type Minio struct {
}

func (m *Minio) Init(_ context.Context, _ config.StorageConfig) error {
	return fmt.Errorf("minio storage is not supported in this build")
}

func (m *Minio) Type() storenum.StorageType {
	return storenum.Minio
}

func (m *Minio) Name() string {
	return ""
}

func (m *Minio) JoinStoragePath(p string) string {
	return strings.TrimPrefix(path.Join("", p), "/")
}

func (m *Minio) Save(_ context.Context, _ io.Reader, _ string) error {
	return fmt.Errorf("minio storage is not supported in this build")
}

func (m *Minio) Exists(_ context.Context, _ string) bool {
	return false
}
