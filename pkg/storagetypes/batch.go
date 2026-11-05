package storagetypes

import "io"

// BatchItem describes one seekable file in a logical batch storage operation.
type BatchItem struct {
	Reader      io.ReadSeeker
	StoragePath string
	Size        int64

	// SourceGroupKey is empty for standalone source messages.
	SourceGroupKey string
	Caption        string
	// PreserveCaption distinguishes an intentionally empty source caption from
	// the storage backend's default caption.
	PreserveCaption bool
	// MetadataJSON is the optional normalized Telegram message metadata.
	MetadataJSON string
	// MetadataGroupKey is derived from metadata grouped_id and is used as a
	// fallback by backends that can preserve source albums.
	MetadataGroupKey string
}
