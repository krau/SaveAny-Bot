package handlers

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/conflictutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func handleConfigCmd(ctx *ext.Context, update *ext.Update) error {
	ctx.Reply(update, ext.ReplyTextString(configMenuText("")), &ext.ReplyOpts{
		Markup: configMenuMarkup(),
	})
	return dispatcher.EndGroups
}

func configMenuText(extra string) string {
	text := i18n.T(i18nk.BotMsgConfigMenuTitle) + "\n\n" + i18n.T(i18nk.BotMsgConfigMenuHint)
	if extra != "" {
		text += "\n\n" + extra
	}
	return text
}

func configMenuMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{
		{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgConfigButtonFilenameStrategy), fmt.Appendf(nil, "%s fnamest", tcbdata.TypeConfig)),
			markup.Callback(i18n.T(i18nk.BotMsgConfigButtonConflictStrategy), fmt.Appendf(nil, "%s conflictst", tcbdata.TypeConfig)),
		}},
		{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgConfigButtonSilent), fmt.Appendf(nil, "%s silent", tcbdata.TypeConfig)),
			markup.Callback(i18n.T(i18nk.BotMsgConfigButtonDefaultStorage), fmt.Appendf(nil, "%s storage", tcbdata.TypeConfig)),
		}},
		{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgConfigButtonTemplate), fmt.Appendf(nil, "%s template", tcbdata.TypeConfig)),
		}},
	}}
}

func handleConfigCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	invaildDataAnswer := func() error {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   i18n.T(i18nk.BotMsgConfigErrorInvalidCallbackData),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	if len(args) < 2 {
		return invaildDataAnswer()
	}
	switch args[1] {
	case "menu":
		return handleConfigMenuCallback(ctx, update)
	case "fnamest":
		return handleConfigFnameSTCallback(ctx, update)
	case "conflictst":
		return handleConfigConflictSTCallback(ctx, update)
	case "silent":
		return handleConfigSilentCallback(ctx, update)
	case "storage":
		return handleConfigStorageCallback(ctx, update)
	case "template":
		return handleConfigTemplateCallback(ctx, update)
	default:
		return invaildDataAnswer()
	}
}

func handleConfigMenuCallback(ctx *ext.Context, update *ext.Update) error {
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     configMenuText(""),
		ReplyMarkup: configMenuMarkup(),
	})
	return dispatcher.EndGroups
}

func handleConfigSilentCallback(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.Silent && user.DefaultStorageID == 0 {
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), i18n.T(i18nk.BotMsgCommonErrorDefaultStorageNotSet)))
		return dispatcher.EndGroups
	}
	user.Silent = !user.Silent
	if err := database.UpdateUser(ctx, user); err != nil {
		return err
	}
	state := i18n.T(i18nk.BotMsgCommonInfoSilentModeOff)
	if user.Silent {
		state = i18n.T(i18nk.BotMsgCommonInfoSilentModeOn)
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     configMenuText(state),
		ReplyMarkup: configMenuMarkup(),
	})
	return dispatcher.EndGroups
}

func handleConfigStorageCallback(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	storages := storage.GetUserStorages(ctx, userID)
	if len(storages) == 0 {
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), i18n.T(i18nk.BotMsgCommonErrorNoAvailableStorage)))
		return dispatcher.EndGroups
	}
	markup, err := msgelem.BuildSetDefaultStorageMarkup(ctx, storages)
	if err != nil {
		return err
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     i18n.T(i18nk.BotMsgCommonPromptSelectDefaultStorage),
		ReplyMarkup: markup,
	})
	return dispatcher.EndGroups
}

func handleConfigTemplateCallback(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	text := i18n.T(i18nk.BotMsgConfigTemplateHelp)
	if user.FilenameTemplate != "" {
		text += "\n\n" + i18n.T(i18nk.BotMsgConfigInfoCurrentTemplatePrefix, map[string]any{
			"Template": user.FilenameTemplate,
		})
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: text,
		ReplyMarkup: &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{{
			Buttons: []tg.KeyboardInlineButton{
				markup.Callback(i18n.T(i18nk.BotMsgConfigButtonBack), fmt.Appendf(nil, "%s menu", tcbdata.TypeConfig)),
			},
		}}},
	})
	return dispatcher.EndGroups
}

func handleConfigFnameSTCallback(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) == 3 {
		selected := args[2]
		st, err := fnamest.ParseFnameST(selected)
		if err != nil {
			return err
		}
		user.FilenameStrategy = st.String()
		if err := database.UpdateUser(ctx, user); err != nil {
			return err
		}
		ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
			ID: update.CallbackQuery.GetMsgID(),
			Message: i18n.T(i18nk.BotMsgConfigInfoFilenameStrategySet, map[string]any{
				"Strategy": fnamest.GetDisplay(st, config.C().Lang),
			}),
		})
		return dispatcher.EndGroups
	}
	opts := fnamest.FnameSTValues()
	buttons := make([]tg.KeyboardInlineButton, 0, len(opts))
	for _, opt := range opts {
		buttons = append(buttons, markup.Callback(
			fnamest.GetDisplay(opt, config.C().Lang),
			fmt.Appendf(nil, "%s %s %s", tcbdata.TypeConfig, "fnamest", opt),
		))
	}
	inlineKeyboard := &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{
		{Buttons: buttons},
	}}
	currentStStr := user.FilenameStrategy
	if currentStStr == "" {
		currentStStr = fnamest.Default.String()
	}
	currentSt, err := fnamest.ParseFnameST(currentStStr)
	if err != nil {
		currentSt = fnamest.Default
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID: update.CallbackQuery.GetMsgID(),
		Message: i18n.T(i18nk.BotMsgConfigPromptSelectFilenameStrategy, map[string]any{
			"Strategy": fnamest.GetDisplay(currentSt, config.C().Lang),
		}),
		ReplyMarkup: inlineKeyboard,
	})
	return dispatcher.EndGroups
}

func handleConfigConflictSTCallback(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) == 3 {
		selected := args[2]
		if !tcbdata.IsConflictStrategy(selected) {
			return fmt.Errorf("invalid conflict strategy: %s", selected)
		}
		user.ConflictStrategy = selected
		if err := database.UpdateUser(ctx, user); err != nil {
			return err
		}
		ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
			ID: update.CallbackQuery.GetMsgID(),
			Message: i18n.T(i18nk.BotMsgConfigInfoConflictStrategySet, map[string]any{
				"Strategy": conflictutil.Display(selected),
			}),
		})
		return dispatcher.EndGroups
	}

	opts := tcbdata.ConflictStrategyValues()
	rows := make([]tg.KeyboardInlineButtonRow, 0, len(opts))
	for _, opt := range opts {
		rows = append(rows, tg.KeyboardInlineButtonRow{
			Buttons: []tg.KeyboardInlineButton{
				markup.Callback(
					conflictutil.Display(opt),
					fmt.Appendf(nil, "%s %s %s", tcbdata.TypeConfig, "conflictst", opt),
				),
			},
		})
	}
	inlineKeyboard := &tg.ReplyInlineMarkup{Rows: rows}
	currentSt := conflictutil.EffectiveStrategy(user)
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID: update.CallbackQuery.GetMsgID(),
		Message: i18n.T(i18nk.BotMsgConfigPromptSelectConflictStrategy, map[string]any{
			"Strategy": conflictutil.Display(currentSt),
		}),
		ReplyMarkup: inlineKeyboard,
	})
	return dispatcher.EndGroups
}

func handleTemplateCmd(ctx *ext.Context, update *ext.Update) error {
	userID := responsibleUserID(update)
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	args := strings.Fields(string(update.EffectiveMessage.Text))
	if len(args) <= 1 {
		text := i18n.T(i18nk.BotMsgConfigTemplateHelp, nil)
		if user.FilenameTemplate != "" {
			text += "\n\n" + i18n.T(i18nk.BotMsgConfigInfoCurrentTemplatePrefix, map[string]any{
				"Template": user.FilenameTemplate,
			})
		}
		ctx.Reply(update, ext.ReplyTextString(text), nil)
		return dispatcher.EndGroups
	}
	newTmpl := strings.Join(args[1:], " ")
	_, err = template.New("filename").Parse(newTmpl)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgConfigErrorInvalidTemplate, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	user.FilenameTemplate = newTmpl
	if err := database.UpdateUser(ctx, user); err != nil {
		return err
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgConfigInfoTemplateUpdated, nil)), nil)
	return dispatcher.EndGroups
}
