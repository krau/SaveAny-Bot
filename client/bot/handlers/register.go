package handlers

import (
	"regexp"

	"github.com/gotd/td/tg"
	sabotfilters "github.com/krau/SaveAny-Bot/client/bot/handlers/utils/filters"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/dispatcher/handlers"
	"github.com/krau/mygotg/dispatcher/handlers/filters"
	"github.com/krau/mygotg/ext"
)

type DescCommandHandler struct {
	Cmd     string
	Desc    i18nk.Key
	handler func(ctx *ext.Context, u *ext.Update) error
}

var CommandHandlers = []DescCommandHandler{
	{"start", i18nk.BotMsgCmdStart, handleHelpCmd},
	{"save", i18nk.BotMsgCmdSave, handleSilentMode(handleSaveCmd, handleSilentSaveReplied)},
	{"download", i18nk.BotMsgCmdDownload, handleDownloadCmd},
	{"transfer", i18nk.BotMsgCmdTransfer, handleTransferCmd},
	{"tasks", i18nk.BotMsgCmdTasks, handleTasksCmd},
	{"config", i18nk.BotMsgCmdConfig, handleConfigCmd},
	{"template", i18nk.BotMsgCmdTemplate, handleTemplateCmd},
	{"dirs", i18nk.BotMsgCmdDirs, handleDirCmd},
	{"rules", i18nk.BotMsgCmdRules, handleRuleCmd},
	{"watch", i18nk.BotMsgCmdWatch, handleWatchCmd},
	{"parser", i18nk.BotMsgCmdParser, handleParserCmd},
	{"update", i18nk.BotMsgCmdUpdate, handleUpdateCmd},
}

// groupGate admits private chats, basic groups, and supergroups. Broadcast
// channels and updates without a resolvable human sender are rejected.
func groupGate(_ *ext.Context, u *ext.Update) error {
	message := u.EffectiveMessage
	if message == nil || message.Out {
		return dispatcher.EndGroups
	}
	switch message.PeerID.(type) {
	case *tg.PeerUser, *tg.PeerChat:
	case *tg.PeerChannel:
		// The channel entity can be absent from short updates. Broadcast
		// channel posts are rejected by the sender check below because their
		// sender is the channel, not a user.
	default:
		return dispatcher.EndGroups
	}
	if responsibleUserID(u) == 0 {
		return dispatcher.EndGroups
	}
	return dispatcher.ContinueGroups
}

func Register(disp dispatcher.Dispatcher) {
	disp.AddHandler(handlers.NewMessage(filters.Message.All, groupGate))
	disp.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
	for _, info := range CommandHandlers {
		disp.AddHandler(handlers.NewCommand(info.Cmd, info.handler))
	}
	// Group chats only react to explicit commands and callback buttons. Plain
	// messages would otherwise save every album or link posted in the group
	// once privacy mode is disabled.
	disp.AddHandler(handlers.NewMessage(filters.Message.All, func(ctx *ext.Context, u *ext.Update) error {
		if isGroupChat(u) {
			return dispatcher.EndGroups
		}
		return dispatcher.ContinueGroups
	}))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("update"), withPermission(handleUpdateCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeAdd), withPermission(handleAddCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeSetDefault), withPermission(handleSetDefaultCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeCancel), withPermission(handleCancelCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeConfig), withPermission(handleConfigCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeEngine), withPermission(handleEngineCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeTask), withPermission(handleTaskCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeRetry), withPermission(handleRetryCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeWatch), withPermission(handleWatchCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeDir), withPermission(handleDirCallback)))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeRule), withPermission(handleRuleCallback)))
	disp.AddHandler(handlers.NewMessage(sabotfilters.RegexUrl(regexp.MustCompile(re.TgMessageLinkRegexString)), skipLinkForMedia(handleSilentMode(handleMessageLink, handleSilentSaveLink))))
	disp.AddHandler(handlers.NewMessage(sabotfilters.RegexUrl(regexp.MustCompile(re.TelegraphUrlRegexString)), skipLinkForMedia(handleSilentMode(handleTelegraphUrlMessage, handleSilentSaveTelegraph))))
	disp.AddHandler(handlers.NewMessage(filters.Message.Media, handleSilentMode(handleMediaMessage, handleSilentSaveMedia)))
	disp.AddHandler(handlers.NewMessage(filters.Message.Text, handleSilentMode(handleTextMessage, handleSilentSaveText)))

}
