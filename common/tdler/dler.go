package tdler

import (
	"github.com/gotd/td/telegram/downloader"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type Client interface {
	downloader.Client
}

func NewDownloader(client Client, file tfile.TGFile) *downloader.Builder {
	return downloader.NewDownloader().WithPartSize(tglimit.MaxPartSize).
		Download(client, file.Location()).WithThreads(dlutil.BestThreads(file.Size(), config.Cfg.Threads))
}
