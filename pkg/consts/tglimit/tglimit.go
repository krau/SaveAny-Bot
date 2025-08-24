package tglimit

import (
	"github.com/gotd/td/telegram/uploader"
)

const (
	MaxPartSize       = 1024 * 1024
	MaxUploadPartSize = uploader.MaximumPartSize
	MaxPhotoSize      = 10 * 1024 * 1024
)
