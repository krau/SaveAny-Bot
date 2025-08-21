package shortcut

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/common/utils/tphutil"
	"github.com/krau/SaveAny-Bot/core"
	tphtask "github.com/krau/SaveAny-Bot/core/tasks/telegraph"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func CreateAndAddtelegraphWithEdit(
	ctx *ext.Context,
	userID int64,
	tphpage *telegraph.Page,
	dirPath string, // unescaped ph path for file storage
	pics []string,
	stor storage.Storage,
	trackMsgID int) error {
		
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	task := tphtask.NewTask(xid.New().String(),
		injectCtx,
		tphpage.Path,
		pics,
		stor,
		stor.JoinStoragePath(dirPath),
		tphutil.DefaultClient(),
		tphtask.NewProgress(trackMsgID, userID),
	)
	if err := core.AddTask(injectCtx, task); err != nil {
		log.FromContext(ctx).Errorf("Failed to add task: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "任务添加失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	text, entities := msgelem.BuildTaskAddedEntities(ctx, tphpage.Title, core.GetLength(ctx))
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:       trackMsgID,
		Message:  text,
		Entities: entities,
	})
	return dispatcher.EndGroups
}
