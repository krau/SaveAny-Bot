package core

import "github.com/gotd/td/telegram/downloader"

var Downloader *downloader.Downloader

func init() {
	Downloader = downloader.NewDownloader().WithPartSize(1024 * 1024)
}
