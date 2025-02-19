package bootstrap

import (
	"fmt"
	"os"

	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage"
)

func InitAll() {
	if err := config.Init(); err != nil {
		fmt.Println("Failed to init config: ", err)
		os.Exit(1)
	}
	logger.InitLogger()
	logger.L.Info("Starting SaveAny-Bot...")
	storage.LoadStorages()
	common.Init()
	dao.Init()
	bot.Init()
}
