package bot

import (
	"os"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
)

var (
	Client *telegram.Client
)

func Init() {
	logger.L.Debug("Initializing bot...")
	var err error
	Client, err = telegram.NewClient(telegram.ClientConfig{
		AppID:    config.Cfg.Telegram.AppID,
		AppHash:  config.Cfg.Telegram.AppHash,
		LogLevel: telegram.LogInfo,
	})
	if err != nil {
		logger.L.Fatal("Failed to create telegram client: ", err)
		os.Exit(1)
	}
	if err := Client.LoginBot(config.Cfg.Telegram.Token); err != nil {
		logger.L.Fatal("Failed to login bot: ", err)
		os.Exit(1)
	}
	logger.L.Info("Bot logged in")
	_, err = Client.BotsSetBotCommands(&telegram.BotCommandScopeDefault{}, "", []*telegram.BotCommand{
		{Command: "start", Description: "开始使用"},
		{Command: "help", Description: "显示帮助"},
		{Command: "silent", Description: "静默模式"},
		{Command: "storage", Description: "设置默认存储位置"},
		{Command: "save", Description: "保存所回复文件"},
	})
	if err != nil {
		logger.L.Errorf("Failed to set bot commands: ", err)
	}
	logger.L.Info("Bot initialized")
}

func Run() {
	if Client == nil {
		Init()
	}

	Client.On("command:start", Start, telegram.FilterPrivate, telegram.FilterChats(config.Cfg.Telegram.Admins...))
	Client.On("command:help", Help, telegram.FilterPrivate, telegram.FilterChats(config.Cfg.Telegram.Admins...))
	Client.On("command:silent", ChangeSilentMode, telegram.FilterPrivate, telegram.FilterChats(config.Cfg.Telegram.Admins...))
	Client.On("command:storage", SetDefaultStorage, telegram.FilterPrivate, telegram.FilterChats(config.Cfg.Telegram.Admins...))
	Client.On("command:save", SaveCmd, telegram.FilterPrivate, telegram.FilterChats(config.Cfg.Telegram.Admins...))
	Client.On(telegram.OnMessage, HandleFileMessage, telegram.FilterPrivate, telegram.FilterChats(config.Cfg.Telegram.Admins...), telegram.FilterMedia)
	Client.On("callback:add", AddToQueue)

	Client.Idle()
}
