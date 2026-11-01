package handlers

import (
	"strings"

	"github.com/charmbracelet/log"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/core/retry"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

func handleRetryCmd(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(update.EffectiveMessage.Text)
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRetryUsage, nil)), nil)
		return dispatcher.EndGroups
	}
	taskID := args[1]
	candidates := []*ext.Context{ctx}
	if uc := userclient.GetCtx(); uc != nil {
		candidates = append(candidates, uc)
	}
	newID, count, err := retry.FailedTask(ctx, candidates, taskID)
	if err != nil {
		log.FromContext(ctx).Errorf("failed to retry task %s: %v", taskID, err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRetryErrorFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRetryInfoStarted, map[string]any{
		"TaskID": newID,
		"Count":  count,
	})), nil)
	return dispatcher.EndGroups
}
