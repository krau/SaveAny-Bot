package handlers

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func handleCancelCallback(ctx *ext.Context, update *ext.Update) error {
	dataParts := strings.Split(string(update.CallbackQuery.Data), " ")
	if len(dataParts) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	taskid := dataParts[1]
	if err := core.CancelTask(ctx, taskid); err != nil {
		log.FromContext(ctx).Errorf("Failed to cancel task %s: %v", taskid, err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), i18n.T(i18nk.BotMsgCancelErrorCancelFailed, map[string]any{
			"Error": err.Error(),
		})))
		return dispatcher.EndGroups
	}

	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: i18n.T(i18nk.BotMsgCancelInfoCancellingTask, nil),
	})

	return dispatcher.EndGroups
}
