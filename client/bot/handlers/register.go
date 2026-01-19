package handlers

import (
	"regexp"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	sabotfilters "github.com/krau/SaveAny-Bot/client/bot/handlers/utils/filters"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
)

type DescCommandHandler struct {
	Cmd     string
	Desc    i18nk.Key
	handler func(ctx *ext.Context, u *ext.Update) error
}

var CommandHandlers = []DescCommandHandler{
	{"start", i18nk.BotMsgCmdStart, handleHelpCmd},
	{"silent", i18nk.BotMsgCmdSilent, handleSilentCmd},
	{"storage", i18nk.BotMsgCmdStorage, handleStorageCmd},
	{"dir", i18nk.BotMsgCmdDir, handleDirCmd},
	{"rule", i18nk.BotMsgCmdRule, handleRuleCmd},
	{"save", i18nk.BotMsgCmdSave, handleSilentMode(handleSaveCmd, handleSilentSaveReplied)},
	{"dl", i18nk.BotMsgCmdDl, handleDlCmd},
	{"aria2dl", i18nk.BotMsgCmdAria2dl, handleAria2DlCmd},
	{"ytdlp", i18nk.BotMsgCmdYtdlp, handleYtdlpCmd},
	{"transfer", i18nk.BotMsgCmdTransfer, handleTransferCmd},
	{"task", i18nk.BotMsgCmdTask, handleTaskCmd},
	{"cancel", i18nk.BotMsgCmdCancel, handleCancelCmd},
	{"config", i18nk.BotMsgCmdConfig, handleConfigCmd},
	{"fnametmpl", i18nk.BotMsgCmdFnametmpl, handleConfigFnameTmpl},
	{"help", i18nk.BotMsgCmdHelp, handleHelpCmd},
	{"parser", i18nk.BotMsgCmdParser, handleParserCmd},
	{"watch", i18nk.BotMsgCmdWatch, handleWatchCmd},
	{"unwatch", i18nk.BotMsgCmdUnwatch, handleUnwatchCmd},
	{"lswatch", i18nk.BotMsgCmdLswatch, handleLswatchCmd},
	{"syncpeers", i18nk.BotMsgCmdSyncpeers, handleSyncpeersCmd},
	{"update", i18nk.BotMsgCmdUpdate, handleUpdateCmd},
}

func Register(disp dispatcher.Dispatcher) {
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChannel), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChat), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
	for _, info := range CommandHandlers {
		disp.AddHandler(handlers.NewCommand(info.Cmd, info.handler))
	}
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("update"), handleUpdateCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeAdd), handleAddCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeSetDefault), handleSetDefaultCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeCancel), handleCancelCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeConfig), handleConfigCallback))
	disp.AddHandler(handlers.NewMessage(sabotfilters.RegexUrl(regexp.MustCompile(re.TgMessageLinkRegexString)), handleSilentMode(handleMessageLink, handleSilentSaveLink)))
	disp.AddHandler(handlers.NewMessage(sabotfilters.RegexUrl(regexp.MustCompile(re.TelegraphUrlRegexString)), handleSilentMode(handleTelegraphUrlMessage, handleSilentSaveTelegraph)))
	disp.AddHandler(handlers.NewMessage(filters.Message.Media, handleSilentMode(handleMediaMessage, handleSilentSaveMedia)))
	disp.AddHandler(handlers.NewMessage(filters.Message.Text, handleSilentMode(handleTextMessage, handleSilentSaveText)))

	if config.C().Telegram.Userbot.Enable {
		go listenMediaMessageEvent(userclient.GetMediaMessageCh())
	}
}
