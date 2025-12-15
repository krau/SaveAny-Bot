package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/styling"
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
			ctx.Reply(update, ext.ReplyTextString("用法: /tasks cancel <task_id>"), nil)
			return dispatcher.EndGroups
		}
		taskID := args[2]
		if err := core.CancelTask(ctx, taskID); err != nil {
			logger.Errorf("取消任务 %s 失败: %v", taskID, err)
			ctx.Reply(update, ext.ReplyTextString("取消任务失败: "+err.Error()), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextStyledTextArray([]styling.StyledTextOption{
			styling.Plain("已请求取消任务: "),
			styling.Code(taskID),
		}), nil)
	default:
		ctx.Reply(update, ext.ReplyTextString("用法: /tasks [running|queued|cancel <task_id>]"), nil)
	}
	return dispatcher.EndGroups
}

func showRunningTasks(ctx *ext.Context, update *ext.Update) {
	tasks := core.GetRunningTasks(ctx)
	if len(tasks) == 0 {
		ctx.Reply(update, ext.ReplyTextString("当前没有正在运行的任务"), nil)
		return
	}
	opts := make([]styling.StyledTextOption, 0, 2+len(tasks)*4)
	opts = append(opts,
		styling.Bold("当前正在运行的任务:"),
		styling.Plain(fmt.Sprintf("\n总数: %d\n", len(tasks))),
	)
	for _, t := range tasks {
		created := t.Created.In(time.Local).Format("2006-01-02 15:04:05")
		status := "运行中"
		if t.Cancelled {
			status = "已请求取消"
		}
		opts = append(opts,
			styling.Plain("\nID: "),
			styling.Code(t.ID),
			styling.Plain("\n名称: "),
			styling.Code(t.Title),
			styling.Plain("\n创建时间: "),
			styling.Code(created),
			styling.Plain("\n状态: "),
			styling.Code(status),
		)
	}
	ctx.Reply(update, ext.ReplyTextStyledTextArray(opts), nil)
}

func showQueuedTasks(ctx *ext.Context, update *ext.Update) {
	tasks := core.GetQueuedTasks(ctx)
	if len(tasks) == 0 {
		ctx.Reply(update, ext.ReplyTextString("当前没有排队中的任务"), nil)
		return
	}
	opts := make([]styling.StyledTextOption, 0, 2+len(tasks)*3)
	opts = append(opts,
		styling.Bold("当前排队中的任务:"),
		styling.Plain(fmt.Sprintf("\n总数: %d\n", len(tasks))),
	)
	for _, t := range tasks {
		created := t.Created.In(time.Local).Format("2006-01-02 15:04:05")
		status := "排队中"
		if t.Cancelled {
			status = "已请求取消"
		}
		opts = append(opts,
			styling.Plain("\nID: "),
			styling.Code(t.ID),
			styling.Plain("\n名称: "),
			styling.Code(t.Title),
			styling.Plain("\n创建时间: "),
			styling.Code(created),
			styling.Plain("\n状态: "),
			styling.Code(status),
		)
		if len(tasks) > 10 {
			opts = append(opts, styling.Plain("\n...\n只显示前 10 个任务, 共 "+fmt.Sprintf("%d", len(tasks))+" 个任务"))
			break
		}
	}
	ctx.Reply(update, ext.ReplyTextStyledTextArray(opts), nil)
}
