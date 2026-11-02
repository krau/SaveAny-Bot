package tdler

import (
	"context"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/relay"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/rpclimit"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

func NewDownloader(ctx context.Context, file tfile.TGFile) (*downloader.Builder, func(context.Context)) {
	file, cleanup := relay.Prepare(ctx, file)
	return newDownloader(file), cleanup
}

func newDownloader(file tfile.TGFile) *downloader.Builder {
	return downloader.NewDownloader().WithPartSize(tglimit.MaxPartSize).
		Download(eofAwareClient{Client: rpclimit.WrapDownloader(file.Dler()), size: file.Size()}, file.Location()).
		WithThreads(downloadThreads(file))
}

func downloadThreads(file tfile.TGFile) int {
	threads := dlutil.BestThreads(file.Size(), config.C().Threads)
	if scalable, ok := file.(interface{ WorkerCount() int }); ok {
		threads = max(threads, min(scalable.WorkerCount()*2, 16))
	}
	return threads
}

// eofAwareClient answers upload.getFile requests at or past the end of the
// file with an empty chunk. gotd's downloader is size-unaware: for files
// whose size is an exact multiple of the part size it issues one final
// request at offset == size and expects an empty chunk, but Telegram rejects
// it with 400 OFFSET_INVALID and the whole download fails.
type eofAwareClient struct {
	downloader.Client
	size int64
}

func (c eofAwareClient) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	if c.size > 0 && req.Offset >= c.size {
		return &tg.UploadFile{}, nil
	}
	return c.Client.UploadGetFile(ctx, req)
}
