package shortcut

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/aria2dl"
	"github.com/krau/SaveAny-Bot/pkg/aria2"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func CreateAndAddAria2TaskWithEdit(ctx *ext.Context, stor storage.Storage, dirPath string, uris []string, aria2Client *aria2.Client, msgID int, userID int64) error {
	logger := log.FromContext(ctx)
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)

	// Now add to aria2 after user selected storage
	logger.Infof("Adding download to aria2, uris type: %T, value: %+v", uris, uris)

	// Ensure uris is valid
	if len(uris) == 0 {
		logger.Error("URIs list is empty")
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      msgID,
			Message: i18n.T(i18nk.BotMsgDlErrorNoValidLinks, nil),
		})
		return dispatcher.EndGroups
	}

	gid, err := aria2Client.AddURI(ctx, uris, nil)
	if err != nil {
		logger.Errorf("Failed to add aria2 download: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: msgID,
			Message: i18n.T(i18nk.BotMsgAria2ErrorAddingAria2Download, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	logger.Infof("Aria2 download added with GID: %s", gid)

	// Create task with the GID
	task := aria2dl.NewTask(xid.New().String(), injectCtx, gid, uris, aria2Client, stor, dirPath, aria2dl.NewProgress(msgID, userID))
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("Failed to add task: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: msgID,
			Message: i18n.T(i18nk.BotMsgCommonErrorTaskAddFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:      msgID,
		Message: i18n.T(i18nk.BotMsgCommonInfoTaskAdded, nil),
	})
	return dispatcher.EndGroups
}
