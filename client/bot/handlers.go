package bot

// import (
// 	"github.com/celestix/gotgproto/dispatcher"
// 	"github.com/celestix/gotgproto/dispatcher/handlers"
// 	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
// 	"github.com/celestix/gotgproto/ext"
// 	"github.com/krau/SaveAny-Bot/common"
// )

// func RegisterHandlers(disp dispatcher.Dispatcher) {
// 	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChannel), func(ctx *ext.Context, u *ext.Update) error {
// 		return dispatcher.EndGroups
// 	}))
// 	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChat), func(ctx *ext.Context, u *ext.Update) error {
// 		return dispatcher.EndGroups
// 	}))
// 	disp.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
// 	disp.AddHandler(handlers.NewCommand("start", start))
// 	disp.AddHandler(handlers.NewCommand("help", help))
// 	disp.AddHandler(handlers.NewCommand("silent", silent))
// 	disp.AddHandler(handlers.NewCommand("storage", storageCmd))
// 	disp.AddHandler(handlers.NewCommand("save", saveCmd))
// 	disp.AddHandler(handlers.NewCommand("dir", dirCmd))
// 	disp.AddHandler(handlers.NewCommand("rule", ruleCmd))
// 	linkRegexFilter, err := filters.Message.Regex(linkRegexString)
// 	if err != nil {
// 		common.Log.Panicf("创建正则表达式过滤器失败: %s", err)
// 	}
// 	disp.AddHandler(handlers.NewMessage(linkRegexFilter, handleLinkMessage))
// 	telegraphUrlRegexFilter, err := filters.Message.Regex(TelegraphUrlRegexString)
// 	if err != nil {
// 		common.Log.Panicf("创建 Telegraph URL 正则表达式过滤器失败: %s", err)
// 	}
// 	disp.AddHandler(handlers.NewMessage(telegraphUrlRegexFilter, handleTelegraph))
// 	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("add"), AddToQueue))
// 	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("set_default"), setDefaultStorage))
// 	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("cancel"), cancelTask))
// 	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("send_here"), sendFileToTelegram))
// 	disp.AddHandler(handlers.NewMessage(filters.Message.Media, handleFileMessage))
// }
