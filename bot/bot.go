package bot

import (
	"os"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
)

var (
	Bot *telego.Bot
)

func Init() {
	logger.L.Debug("Initializing bot...")
	var err error
	Bot, err = telego.NewBot(
		config.Cfg.Telegram.Token,
		telego.WithDefaultLogger(false, true),
		telego.WithAPIServer(config.Cfg.Telegram.API),
	)
	if err != nil {
		logger.L.Fatal("Failed to create bot: ", err)
		os.Exit(1)
	}
	Bot.SetMyCommands(&telego.SetMyCommandsParams{
		Commands: []telego.BotCommand{
			{Command: "start", Description: "开始使用"},
			{Command: "help", Description: "显示帮助"},
			{Command: "silent", Description: "静默模式"},
			{Command: "storage", Description: "设置默认存储位置"},
			{Command: "save", Description: "保存文件"},
			{Command: "clean", Description: "清除文件记录"},
		},
	})
	logger.L.Debug("Bot initialized")
}

func Run() {
	if Bot == nil {
		Init()
	}
	logger.L.Info("Start polling...")
	updates, err := Bot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
		Offset: -1,
		AllowedUpdates: []string{
			telego.MessageUpdates,
			telego.CallbackQueryUpdates,
		},
	})
	if err != nil {
		logger.L.Fatal("Failed to start polling: ", err)
		os.Exit(1)
	}
	botHandler, err := telegohandler.NewBotHandler(Bot, updates)
	if err != nil {
		logger.L.Fatal("Failed to create bot handler: ", err)
		os.Exit(1)
	}
	defer botHandler.Stop()
	defer Bot.StopLongPolling()

	botHandler.Use(telegohandler.PanicRecovery())
	baseGroup := botHandler.BaseGroup()

	registerHandlers(baseGroup)

	botHandler.Start()
}

func registerHandlers(hg *telegohandler.HandlerGroup) {
	msgGroup := hg.Group(telegohandler.AnyMessage())
	msgGroup.Use(func(bot *telego.Bot, update telego.Update, next telegohandler.Handler) {
		if !slice.Contain(config.Cfg.Telegram.Admins, update.Message.From.ID) {
			bot.SendMessage(telegoutil.Message(update.Message.Chat.ChatID(), "抱歉, 该 Bot 为个人使用设计, 您可以部署自己的 SaveAnyBot 实例: https://github.com/krau/SaveAny-Bot"))
			return
		}
		next(bot, update)
	})

	msgGroup.HandleMessageCtx(Start, telegohandler.CommandEqual("start"))
	msgGroup.HandleMessageCtx(Help, telegohandler.CommandEqual("help"))
	msgGroup.HandleMessageCtx(ChangeSilentMode, telegohandler.CommandEqual("silent"))
	msgGroup.HandleMessageCtx(SetDefaultStorage, telegohandler.CommandEqual("storage"))
	msgGroup.HandleMessageCtx(SaveFile, telegohandler.CommandEqual("save"))
	msgGroup.HandleMessageCtx(CleanReceivedFile, telegohandler.CommandEqual("clean"))

	msgGroup.HandleMessageCtx(HandleFileMessage, func(update telego.Update) bool {
		return update.Message.Document != nil || update.Message.Video != nil || update.Message.Audio != nil
	})

	callbackGroup := hg.Group(telegohandler.AnyCallbackQueryWithMessage())
	callbackGroup.Use(func(bot *telego.Bot, update telego.Update, next telegohandler.Handler) {
		if !slice.Contain(config.Cfg.Telegram.Admins, update.CallbackQuery.From.ID) {
			bot.AnswerCallbackQuery(telegoutil.
				CallbackQuery(update.CallbackQuery.ID).
				WithText("抱歉, 该 Bot 为个人使用设计, 您可以部署自己的 SaveAnyBot 实例: https://github.com/krau/SaveAny-Bot").
				WithShowAlert().
				WithCacheTime(60))
			return
		}
		next(bot, update)
	})

	callbackGroup.HandleCallbackQueryCtx(AddToQueue, telegohandler.CallbackDataPrefix("add"))
}
