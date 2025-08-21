package shortcut

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/parsed"
	"github.com/krau/SaveAny-Bot/pkg/parser"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func CreateAndAddParsedTaskWithEdit(ctx *ext.Context, stor storage.Storage, dirPath string, item *parser.Item, msgID int, userID int64) error {
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	task := parsed.NewTask(xid.New().String(), injectCtx, stor, stor.JoinStoragePath(dirPath), item, parsed.NewProgress(msgID, userID))
	if err := core.AddTask(injectCtx, task); err != nil {
		log.FromContext(ctx).Errorf("Failed to add task: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      msgID,
			Message: "任务添加失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	text, entities := msgelem.BuildTaskAddedEntities(ctx, item.Title, core.GetLength(ctx))
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:       msgID,
		Message:  text,
		Entities: entities,
	})
	return dispatcher.EndGroups
}
