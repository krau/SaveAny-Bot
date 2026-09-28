//go:build no_bubbletea

package upload

import "context"

type uploadModel struct {
}

type UploadProgress struct {
}

func NewUploadProgress(ctx context.Context, fileName string, fileSize int64) *UploadProgress {
	return &UploadProgress{}
}

func (up *UploadProgress) Start() {}

func (up *UploadProgress) UpdateProgress(percent float64) {}

func (up *UploadProgress) SetError(err error) {}

func (up *UploadProgress) Done() {}

func (up *UploadProgress) Wait() {}

func (up *UploadProgress) Quit() {}
