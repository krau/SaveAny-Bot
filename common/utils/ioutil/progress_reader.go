package ioutil

import (
	"io"
	"sync/atomic"
)

var _ io.ReadSeeker = (*ProgressReadSeeker)(nil)

// ProgressReadSeeker wraps an io.ReadSeeker and tracks read progress
type ProgressReadSeeker struct {
	reader     io.ReadSeeker
	total      atomic.Int64
	read       atomic.Int64
	onProgress func(read int64, total int64)
}

// Seek implements io.ReadSeeker.
func (pr *ProgressReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return pr.reader.Seek(offset, whence)
}

// NewProgressReader creates a new ProgressReader
func NewProgressReader(rs io.ReadSeeker, total int64, onProgress func(read int64, total int64)) *ProgressReadSeeker {
	prs := &ProgressReadSeeker{
		reader:     rs,
		total:      atomic.Int64{},
		read:       atomic.Int64{},
		onProgress: onProgress,
	}
	prs.total.Store(total)
	return prs
}

// Read implements io.Reader
func (pr *ProgressReadSeeker) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.read.Add(int64(n))
		read := pr.read.Load()

		if pr.onProgress != nil {
			pr.onProgress(read, pr.total.Load())
		}
	}
	return n, err
}

// Progress returns the current progress as a float64 between 0 and 1
func (pr *ProgressReadSeeker) Progress() float64 {
	if pr.total.Load() <= 0 {
		return 0
	}
	return float64(pr.read.Load()) / float64(pr.total.Load())
}

// Read returns the number of bytes read so far
func (pr *ProgressReadSeeker) BytesRead() int64 {
	return pr.read.Load()
}

// Total returns the total number of bytes
func (pr *ProgressReadSeeker) Total() int64 {
	return pr.total.Load()
}
