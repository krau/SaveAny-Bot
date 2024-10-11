package common

import (
	"path/filepath"
	"time"

	"github.com/imroc/req/v3"
	"github.com/krau/SaveAny-Bot/config"
)

var ReqClient *req.Client

func initClient() {
	ReqClient = req.NewClient().SetOutputDirectory(config.Cfg.Temp.BasePath).SetTimeout(86400 * time.Second)
}

func GetDownloadedFilePath(filename string) string {
	return filepath.Join(config.Cfg.Temp.BasePath, filename)
}
