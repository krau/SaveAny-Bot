package handlers

import (
	"fmt"
	"strconv"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func checkPermission(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	if !slice.Contain(config.Cfg.GetUsersID(), userID) {
		const noPermissionText string = `
您不在白名单中, 无法使用此 Bot.
您可以部署自己的实例: https://github.com/krau/SaveAny-Bot
`
		ctx.Reply(update, ext.ReplyTextString(noPermissionText), nil)
		return dispatcher.EndGroups
	}

	return dispatcher.ContinueGroups
}

func handleSilentSaveMedia(next func(*ext.Context, *ext.Update) error) func(*ext.Context, *ext.Update) error {
	return func(ctx *ext.Context, update *ext.Update) error {
		// TODO: refactor to reduce code duplication
		userID := update.GetUserChat().GetID()
		user, err := database.GetUserByChatID(ctx, userID)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString("获取用户信息失败: "+err.Error()), nil)
			return dispatcher.EndGroups
		}
		if !user.Silent {
			return next(ctx, update)
		}
		if user.DefaultStorage == "" {
			ctx.Reply(update, ext.ReplyTextString("您已开启静默模式, 但未设置默认存储端, 请先使用 /storage 设置"), nil)
			return next(ctx, update)
		}
		stor, err := storage.GetStorageByUserIDAndName(ctx, userID, user.DefaultStorage)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString("获取默认存储失败: "+err.Error()), nil)
			return dispatcher.EndGroups
		}
		logger := log.FromContext(ctx)
		message := update.EffectiveMessage.Message
		logger.Debugf("Got media: %s", message.Media.TypeName())
		media := message.Media
		supported := func(media tg.MessageMediaClass) bool {
			switch media.(type) {
			case *tg.MessageMediaDocument, *tg.MessageMediaPhoto:
				return true
			default:
				return false
			}
		}(media)
		if !supported {
			return dispatcher.EndGroups
		}
		msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
		if err != nil {
			logger.Errorf("回复失败: %s", err)
			return dispatcher.EndGroups
		}
		genFilename := tgutil.GenFileNameFromMessage(*message)
		if genFilename == "" {
			genFilename = xid.New().String()
		}

		file, err := tfile.FromMedia(media, tfile.WithNameIfEmpty(genFilename))
		if err != nil {
			logger.Errorf("获取文件失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
			return dispatcher.EndGroups
		}
		storagePath := stor.JoinStoragePath(file.Name())
		injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
		taskid := xid.New().String()
		task, err := tftask.NewTGFileTask(taskid, injectCtx, file, ctx.Raw, stor, storagePath, tftask.NewProgressTrack(
			msg.ID,
			update.GetUserChat().GetID()))
		if err != nil {
			ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
				QueryID:   update.CallbackQuery.GetQueryID(),
				Alert:     true,
				Message:   "任务创建失败: " + err.Error(),
				CacheTime: 5,
			})
			return dispatcher.EndGroups
		}
		if err := core.AddTask(injectCtx, task); err != nil {
			ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
				QueryID:   update.CallbackQuery.GetQueryID(),
				Alert:     true,
				Message:   "任务添加失败: " + err.Error(),
				CacheTime: 5,
			})
			return dispatcher.EndGroups
		}
		entityBuilder := entity.Builder{}
		var entities []tg.MessageEntityClass
		length := core.GetLength(injectCtx)
		text := fmt.Sprintf("已添加到任务队列\n文件名: %s\n当前排队任务数: %d", file.Name(), length)
		if err := styling.Perform(&entityBuilder,
			styling.Plain("已添加到任务队列\n文件名: "),
			styling.Code(file.Name()),
			styling.Plain("\n当前排队任务数: "),
			styling.Bold(strconv.Itoa(length)),
		); err != nil {
			log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
		} else {
			text, entities = entityBuilder.Complete()
		}
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:       update.CallbackQuery.GetMsgID(),
			Message:  text,
			Entities: entities,
		})

		return dispatcher.EndGroups
	}
}
