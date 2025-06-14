package ioutil

import "io"

type ProgressWriterAt struct {
	wrAt    io.WriterAt
	onWrite func(n int)
}

func (p *ProgressWriterAt) WriteAt(buf []byte, off int64) (n int, err error) {
	n, err = p.wrAt.WriteAt(buf, off)
	if n > 0 {
		p.onWrite(n)
	}
	return
}

func NewProgressWriterAt(
	wrAt io.WriterAt,
	onWrite func(n int),
) *ProgressWriterAt {
	return &ProgressWriterAt{
		wrAt:    wrAt,
		onWrite: onWrite,
	}
}

type ProgressWriter struct {
	wr      io.Writer
	onWrite func(n int)
}

func (p *ProgressWriter) Write(buf []byte) (n int, err error) {
	n, err = p.wr.Write(buf)
	if n > 0 {
		p.onWrite(n)
	}
	return
}

func NewProgressWriter(
	wr io.Writer,
	onWrite func(n int),
) *ProgressWriter {
	return &ProgressWriter{
		wr:      wr,
		onWrite: onWrite,
	}
}
