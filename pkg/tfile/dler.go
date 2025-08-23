package tfile

import (
	"github.com/gotd/td/telegram/downloader"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
)

func NewDownloader(file TGFile) *downloader.Builder {
	return downloader.NewDownloader().WithPartSize(tglimit.MaxPartSize).
		Download(file.Dler(), file.Location()).WithThreads(dlutil.BestThreads(file.Size(), config.C().Threads))
}
