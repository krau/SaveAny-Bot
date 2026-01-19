package handlers

import (
	"strings"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/core"
)

func handleTaskCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Fields(update.EffectiveMessage.Text)
	if len(args) == 1 {
		showRunningTasks(ctx, update)
		return dispatcher.EndGroups
	}

	switch args[1] {
	case "running", "run", "r":
		showRunningTasks(ctx, update)
	case "queued", "queue", "q", "waiting":
		showQueuedTasks(ctx, update)
	case "cancel", "c":
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTasksUsageCancel)), nil)
			return dispatcher.EndGroups
		}
		taskID := args[2]
		if err := core.CancelTask(ctx, taskID); err != nil {
			logger.Errorf("Failed to cancel task %s: %v", taskID, err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTasksCancelFailed, map[string]any{"Error": err.Error()})), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextStyledTextArray([]styling.StyledTextOption{
			styling.Plain(i18n.T(i18nk.BotMsgTasksCancelRequestedPrefix)),
			styling.Code(taskID),
		}), nil)
	default:
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTasksUsage)), nil)
	}
	return dispatcher.EndGroups
}

func showRunningTasks(ctx *ext.Context, update *ext.Update) {
	tasks := core.GetRunningTasks(ctx)
	if len(tasks) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTasksRunningEmpty)), nil)
		return
	}
	opts := make([]styling.StyledTextOption, 0, 2+len(tasks)*4)
	opts = append(opts,
		styling.Bold(i18n.T(i18nk.BotMsgTasksRunningTitle)),
		styling.Plain(i18n.T(i18nk.BotMsgTasksTotalPrefix, map[string]any{"Count": len(tasks)})),
	)
	for _, t := range tasks {
		created := t.Created.In(time.Local).Format("2006-01-02 15:04:05")
		status := i18n.T(i18nk.BotMsgTasksStatusRunning)
		if t.Cancelled {
			status = i18n.T(i18nk.BotMsgTasksStatusCancelRequested)
		}
		opts = append(opts,
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldId)),
			styling.Code(t.ID),
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldTitle)),
			styling.Code(t.Title),
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldCreated)),
			styling.Code(created),
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldStatus)),
			styling.Code(status),
		)
	}
	ctx.Reply(update, ext.ReplyTextStyledTextArray(opts), nil)
}

func showQueuedTasks(ctx *ext.Context, update *ext.Update) {
	tasks := core.GetQueuedTasks(ctx)
	if len(tasks) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTasksQueuedEmpty)), nil)
		return
	}
	opts := make([]styling.StyledTextOption, 0, 2+len(tasks)*3)
	opts = append(opts,
		styling.Bold(i18n.T(i18nk.BotMsgTasksQueuedTitle)),
		styling.Plain(i18n.T(i18nk.BotMsgTasksTotalPrefix, map[string]any{"Count": len(tasks)})),
	)
	for _, t := range tasks {
		created := t.Created.In(time.Local).Format("2006-01-02 15:04:05")
		status := i18n.T(i18nk.BotMsgTasksStatusQueued)
		if t.Cancelled {
			status = i18n.T(i18nk.BotMsgTasksStatusCancelRequested)
		}
		opts = append(opts,
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldId)),
			styling.Code(t.ID),
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldTitle)),
			styling.Code(t.Title),
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldCreated)),
			styling.Code(created),
			styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksFieldStatus)),
			styling.Code(status),
		)
		if len(tasks) > 10 {
			opts = append(opts, styling.Plain("\n"+i18n.T(i18nk.BotMsgTasksTruncatedNote, map[string]any{"Count": len(tasks)})))
			break
		}
	}
	ctx.Reply(update, ext.ReplyTextStyledTextArray(opts), nil)
}
