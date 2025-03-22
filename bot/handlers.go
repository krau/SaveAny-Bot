package bot

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/krau/SaveAny-Bot/common"
)

func RegisterHandlers(dispatcher dispatcher.Dispatcher) {
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
	dispatcher.AddHandler(handlers.NewCommand("start", start))
	dispatcher.AddHandler(handlers.NewCommand("help", help))
	dispatcher.AddHandler(handlers.NewCommand("silent", silent))
	dispatcher.AddHandler(handlers.NewCommand("storage", storageCmd))
	dispatcher.AddHandler(handlers.NewCommand("save", saveCmd))
	dispatcher.AddHandler(handlers.NewCommand("dir", dirCmd))
	linkRegexFilter, err := filters.Message.Regex(linkRegexString)
	if err != nil {
		common.Log.Panicf("创建正则表达式过滤器失败: %s", err)
	}
	dispatcher.AddHandler(handlers.NewMessage(linkRegexFilter, handleLinkMessage))
	telegraphUrlRegexFilter, err := filters.Message.Regex(TelegraphUrlRegexString)
	if err != nil {
		common.Log.Panicf("创建 Telegraph URL 正则表达式过滤器失败: %s", err)
	}
	dispatcher.AddHandler(handlers.NewMessage(telegraphUrlRegexFilter, handleTelegraph))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("add"), AddToQueue))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("set_default"), setDefaultStorage))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("cancel"), cancelTask))
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleFileMessage))
}
