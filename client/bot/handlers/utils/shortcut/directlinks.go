package shortcut

import (
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/directlinks"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
	"github.com/rs/xid"
)

func CreateAndAddDirectTaskWithEdit(ctx *ext.Context, stor storage.Storage, dirPath string, links []string, msgID int, chatID int64) error {
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	task := directlinks.NewTask(xid.New().String(), injectCtx, links, stor, dirPath, directlinks.NewProgress(msgID, chatID))
	if err := core.AddTask(injectCtx, task); err != nil {
		log.FromContext(ctx).Errorf("Failed to add task: %s", err)
		ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
			ID: msgID,
			Message: i18n.T(i18nk.BotMsgCommonErrorTaskAddFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
		Message: i18n.T(i18nk.BotMsgCommonInfoTaskAdded, nil),
	})
	return dispatcher.EndGroups
}
