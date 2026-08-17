package tglimit

import (
	"github.com/gotd/td/telegram/uploader"
)

const (
	MaxPartSize       = 1024 * 1024
	MaxUploadPartSize = uploader.MaximumPartSize
	MaxPhotoSize      = 10 * 1024 * 1024
	// MaxAlbumItems is the Telegram media-album item cap used for batching
	// uploads and lossless video splitting.
	MaxAlbumItems = 10
)
