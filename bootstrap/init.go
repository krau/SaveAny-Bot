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
		fmt.Println("加载配置文件失败: ", err)
		os.Exit(1)
	}
	logger.InitLogger()
	logger.L.Info("正在启动 SaveAny-Bot...")
	storage.LoadStorages()
	common.Init()
	dao.Init()
	bot.Init()
}
