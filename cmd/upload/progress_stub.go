//go:build no_bubbletea

package upload

import "context"

type uploadModel struct {
}

// UploadProgress manages the progress UI for uploads
type UploadProgress struct {
}

// NewUploadProgress creates a new upload progress tracker
func NewUploadProgress(ctx context.Context, fileName string, fileSize int64) *UploadProgress {
	return &UploadProgress{}
}

// Start starts the progress UI in a goroutine and returns immediately
func (up *UploadProgress) Start() {}

// UpdateProgress updates the progress bar with a new percentage (0.0 - 1.0)
func (up *UploadProgress) UpdateProgress(percent float64) {}

// SetError sets an error and quits the progress UI
func (up *UploadProgress) SetError(err error) {}

// Done signals that the upload is complete
func (up *UploadProgress) Done() {}

// Wait waits for the progress UI to finish
func (up *UploadProgress) Wait() {}

// Quit quits the progress UI
func (up *UploadProgress) Quit() {}
