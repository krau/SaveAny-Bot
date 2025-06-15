package handlers

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
)

func Register(disp dispatcher.Dispatcher) {
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChannel), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChat), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
	disp.AddHandler(handlers.NewCommand("start", handleHelpCmd))
	disp.AddHandler(handlers.NewCommand("help", handleHelpCmd))
	disp.AddHandler(handlers.NewCommand("silent", handleSilentCmd))
	disp.AddHandler(handlers.NewCommand("storage", handleStorageCmd))
	disp.AddHandler(handlers.NewCommand("dir", handleDirCmd))
	disp.AddHandler(handlers.NewCommand("rule", handleRuleCmd))
	disp.AddHandler(handlers.NewCommand("save", handleSilentMode(handleSaveCmd, handleSilentSaveReplied)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeAdd), handleAddCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeSetDefault), handleSetDefaultCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("cancel"), handleCancelCallback))
	linkRegexFilter, err := filters.Message.Regex(re.TgMessageLinkRegexString)
	if err != nil {
		panic("failed to create regex filter: " + err.Error())
	}
	disp.AddHandler(handlers.NewMessage(linkRegexFilter, handleSilentMode(handleMessageLink, handleSilentSaveLink)))
	telegraphUrlRegexFilter, err := filters.Message.Regex(re.TelegraphUrlRegexString)
	if err != nil {
		panic("failed to create Telegraph URL regex filter: " + err.Error())
	}
	disp.AddHandler(handlers.NewMessage(telegraphUrlRegexFilter, handleSilentMode(handleTelegraphUrlMessage, handleSilentSaveTelegraph))) // TODO:
	disp.AddHandler(handlers.NewMessage(filters.Message.Media, handleSilentMode(handleMediaMessage, handleSilentSaveMedia)))
}
