package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/metadata"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage/alist"
	"github.com/krau/SaveAny-Bot/storage/local"
	"github.com/krau/SaveAny-Bot/storage/rclone"
	"github.com/krau/SaveAny-Bot/storage/s3"
	"github.com/krau/SaveAny-Bot/storage/telegram"
	"github.com/krau/SaveAny-Bot/storage/webdav"
)

type Storage interface {
	// Init 只应该在创建存储时调用一次
	Init(ctx context.Context, cfg storcfg.StorageConfig) error
	Type() storenum.StorageType
	Name() string
	Save(ctx context.Context, reader io.Reader, storagePath string) error
	// Exists reports whether storagePath exists. Backend/query failures must be
	// returned as an error rather than treated as "not found".
	Exists(ctx context.Context, storagePath string) (bool, error)
}

type StorageCannotStream interface {
	Storage
	CannotStream() string
}

// StorageBatchSaver can preserve relationships between files when saving a
// logical batch, such as a Telegram media album.
type StorageBatchSaver interface {
	Storage
	SaveBatch(ctx context.Context, items []storagetypes.BatchItem) error
}

// StorageBatchProgressSaver reports confirmed upload progress for each item in
// a logical batch. The item index matches the items slice passed to
// SaveBatchWithProgress.
type StorageBatchProgressSaver interface {
	StorageBatchSaver
	SaveBatchWithProgress(
		ctx context.Context,
		items []storagetypes.BatchItem,
		onProgress func(index int, uploaded, total int64),
	) error
}

// StorageProgressSaver reports bytes after the backend has accepted them for
// upload. Backends with native progress support should implement this instead
// of relying on progress inferred from reads of the input stream.
type StorageProgressSaver interface {
	Storage
	SaveWithProgress(
		ctx context.Context,
		reader io.Reader,
		storagePath string,
		onProgress func(uploaded, total int64),
	) error
}

type StorageListable interface {
	Storage
	ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error)
}

// StorageTester verifies that the backend can reach its configured target.
// It is used by storage connection tests before falling back to ListFiles or
// Exists.
type StorageTester interface {
	Storage
	Test(ctx context.Context) error
}

// StorageSidecarWriter lets a backend define how metadata sidecars are stored.
// Telegram storage implements it as a no-op because metadata is consumed during
// the upload instead of being written as a separate chat message.
type StorageSidecarWriter interface {
	Storage
	WriteMetadataSidecar(ctx context.Context, actualPath string, data []byte) error
}

// WriteMetadataSidecar writes a normalized metadata sidecar using the backend
// policy, defaulting to a sidecar file next to the actual saved path.
func WriteMetadataSidecar(ctx context.Context, st Storage, actualPath string, data []byte) error {
	if writer, ok := st.(StorageSidecarWriter); ok {
		return writer.WriteMetadataSidecar(ctx, actualPath, data)
	}
	return st.Save(WithOverwrite(ctx), bytes.NewReader(data), actualPath+metadata.MetaSuffix)
}

// StoragePathSaver returns the actual path assigned by a backend after any
// conflict resolution. Backends without this interface use the requested path.
type StoragePathSaver interface {
	Storage
	SaveWithPath(ctx context.Context, reader io.Reader, storagePath string) (string, error)
}

// SaveWithPath saves a reader and returns the actual storage path when the
// backend can report it.
func SaveWithPath(ctx context.Context, st Storage, reader io.Reader, storagePath string) (string, error) {
	if saver, ok := st.(StoragePathSaver); ok {
		return saver.SaveWithPath(ctx, reader, storagePath)
	}
	return storagePath, st.Save(ctx, reader, storagePath)
}

type StorageReadable interface {
	Storage
	OpenFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error)
}

var _ StorageProgressSaver = (*telegram.Telegram)(nil)
var _ StorageBatchProgressSaver = (*telegram.Telegram)(nil)
var _ StorageTester = (*telegram.Telegram)(nil)
var _ StoragePathSaver = (*local.Local)(nil)
var _ StorageSidecarWriter = (*telegram.Telegram)(nil)

var _ StorageListable = (*alist.Alist)(nil)
var _ StorageReadable = (*alist.Alist)(nil)
var _ StorageListable = (*local.Local)(nil)
var _ StorageReadable = (*local.Local)(nil)
var _ StorageListable = (*rclone.Rclone)(nil)
var _ StorageReadable = (*rclone.Rclone)(nil)
var _ StorageListable = (*webdav.Webdav)(nil)
var _ StorageReadable = (*webdav.Webdav)(nil)

type StorageCloser interface {
	Storage
	Close()
}

type StorageConstructor func() Storage

var storageConstructors = map[storenum.StorageType]StorageConstructor{
	storenum.Alist:    func() Storage { return new(alist.Alist) },
	storenum.Local:    func() Storage { return new(local.Local) },
	storenum.Webdav:   func() Storage { return new(webdav.Webdav) },
	storenum.S3:       func() Storage { return new(s3.S3) },
	storenum.Telegram: func() Storage { return new(telegram.Telegram) },
	storenum.Rclone:   func() Storage { return new(rclone.Rclone) },
}

func NewStorage(ctx context.Context, cfg storcfg.StorageConfig) (Storage, error) {
	constructor, ok := storageConstructors[cfg.GetType()]
	if !ok {
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.GetType())
	}

	storage := constructor()
	if err := storage.Init(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize storage %s: %w", cfg.GetName(), err)
	}

	return storage, nil
}
