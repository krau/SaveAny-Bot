package telegram

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type splitWriter struct {
	baseName    string
	partSize    int64
	currentPart int
	currentSize int64
	currentFile *os.File
	totalParts  int
}

func newSplitWriter(baseName string, partSize int64) *splitWriter {
	return &splitWriter{
		baseName:    baseName,
		partSize:    partSize,
		currentPart: 0,
	}
}

// Write implements io.Writer interface
func (w *splitWriter) Write(p []byte) (n int, err error) {
	written := 0
	for written < len(p) {
		if w.currentFile == nil || w.currentSize >= w.partSize {
			if err := w.nextPart(); err != nil {
				return written, err
			}
		}

		toWrite := int64(len(p) - written)
		remaining := w.partSize - w.currentSize
		if toWrite > remaining {
			toWrite = remaining
		}

		nw, err := w.currentFile.Write(p[written : written+int(toWrite)])
		written += nw
		w.currentSize += int64(nw)

		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func (w *splitWriter) Close() error {
	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

func (w *splitWriter) nextPart() error {
	if w.currentFile != nil {
		if err := w.currentFile.Close(); err != nil {
			return err
		}
	}

	partName := w.partName(w.currentPart)
	file, err := os.Create(partName)
	if err != nil {
		return err
	}

	w.currentFile = file
	w.currentSize = 0
	w.currentPart++
	return nil
}

func (w *splitWriter) partName(partNum int) string {
	// file.zip.001, file.zip.002, ...
	return fmt.Sprintf("%s.zip.%03d", w.baseName, partNum+1)
}

func (w *splitWriter) finalize() error {
	w.totalParts = w.currentPart

	// 如果只有一个分卷,直接重命名为 .zip
	if w.totalParts == 1 {
		oldName := fmt.Sprintf("%s.zip.001", w.baseName)
		newName := fmt.Sprintf("%s.zip", w.baseName)
		return os.Rename(oldName, newName)
	}

	return nil
}

func CreateSplitZip(ctx context.Context, reader io.Reader, size int64, fileName, outputBase string, partSize int64) error {
	outputDir := filepath.Dir(outputBase)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	splitWriter := newSplitWriter(outputBase, partSize)
	defer splitWriter.Close()

	zipWriter := zip.NewWriter(splitWriter)
	defer zipWriter.Close()

	header := &zip.FileHeader{
		Name:     fileName,
		Method:   zip.Store, // just store without compression
		Modified: time.Now(),
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip header: %w", err)
	}

	copied, err := io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	if copied != size {
		return fmt.Errorf("incomplete write: expected %d bytes, got %d bytes", size, copied)
	}
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %w", err)
	}
	if err := splitWriter.Close(); err != nil {
		return fmt.Errorf("failed to close split writer: %w", err)
	}
	if err := splitWriter.finalize(); err != nil {
		return fmt.Errorf("failed to rename split files: %w", err)
	}
	return nil
}
