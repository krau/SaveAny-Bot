package handlers

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
)

func handleConfigCmd(ctx *ext.Context, update *ext.Update) error {
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgConfigPromptSelectOption)), &ext.ReplyOpts{
		Markup: &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: i18n.T(i18nk.BotMsgConfigButtonFilenameStrategy),
							Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeConfig, "fnamest"),
						},
					},
				},
			},
		},
	})
	return dispatcher.EndGroups
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
	case "fnamest":
		return handleConfigFnameSTCallback(ctx, update)
	default:
		return invaildDataAnswer()
	}
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
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: update.CallbackQuery.GetMsgID(),
			Message: i18n.T(i18nk.BotMsgConfigInfoFilenameStrategySet, map[string]any{
				"Strategy": fnamest.GetDisplay(st, config.C().Lang),
			}),
		})
		return dispatcher.EndGroups
	}
	opts := fnamest.FnameSTValues()
	buttons := make([]tg.KeyboardButtonClass, 0, len(opts))
	for _, opt := range opts {
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: fnamest.GetDisplay(opt, config.C().Lang),
			Data: fmt.Appendf(nil, "%s %s %s", tcbdata.TypeConfig, "fnamest", opt),
		})
	}
	markup := &tg.ReplyInlineMarkup{Rows: []tg.KeyboardButtonRow{
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
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID: update.CallbackQuery.GetMsgID(),
		Message: i18n.T(i18nk.BotMsgConfigPromptSelectFilenameStrategy, map[string]any{
			"Strategy": fnamest.GetDisplay(currentSt, config.C().Lang),
		}),
		ReplyMarkup: markup,
	})
	return dispatcher.EndGroups
}

func handleConfigFnameTmpl(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	args := strings.Fields(string(update.EffectiveMessage.Text))
	if len(args) <= 1 {
		text := i18n.T(i18nk.BotMsgConfigFnametmplHelp, nil)
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
