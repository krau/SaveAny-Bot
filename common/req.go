package common

import (
	"path/filepath"

	"github.com/imroc/req/v3"
	"github.com/krau/SaveAny-Bot/config"
)

var ReqClient *req.Client

func initClient() {
	ReqClient = req.NewClient().SetOutputDirectory(config.Cfg.Temp.BasePath)
}

func GetDownloadedFilePath(filename string) string {
	return filepath.Join(config.Cfg.Temp.BasePath, filename)
}
