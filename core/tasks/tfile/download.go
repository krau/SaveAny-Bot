package tfile

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/log"

	"github.com/krau/SaveAny-Bot/common/tdler"
)

// download fetches the file into the cache path, resuming from a partial
// .part download and reusing a complete cache file.
func (t *Task) download(ctx context.Context) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", t.File.Name()))
	if err := tdler.DownloadToCache(ctx, t.File, t.localPath, func(w io.WriterAt) io.WriterAt {
		return newWriterAt(ctx, w, t.Progress, t)
	}); err != nil {
		return err
	}
	logger.Info("File downloaded successfully")
	return nil
}
