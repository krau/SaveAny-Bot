package bootstrap

import (
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
)

func InitAll() {
	config.Init()
	logger.InitLogger()
	logger.L.Info("Starting SaveAny-Bot...")

	common.Init()
	dao.Init()
	bot.Init()
}
