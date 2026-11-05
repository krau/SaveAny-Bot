package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/retry"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

const taskListLimit = 10

func handleTasksCmd(ctx *ext.Context, update *ext.Update) error {
	ctx.Reply(update, ext.ReplyTextString(taskMenuText()), &ext.ReplyOpts{
		Markup: taskMenuMarkup(),
	})
	return dispatcher.EndGroups
}

func taskMenuText() string {
	return i18n.T(i18nk.BotMsgTasksMenuTitle) + "\n\n" + i18n.T(i18nk.BotMsgTasksMenuHint)
}

func taskMenuMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{
		{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgTasksMenuRunning), fmt.Appendf(nil, "%s running", tcbdata.TypeTask)),
			markup.Callback(i18n.T(i18nk.BotMsgTasksMenuQueued), fmt.Appendf(nil, "%s queued", tcbdata.TypeTask)),
		}},
		{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgTasksMenuFailed), fmt.Appendf(nil, "%s failed", tcbdata.TypeTask)),
		}},
	}}
}

func taskBackMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{{
		Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgTasksMenuBack), fmt.Appendf(nil, "%s menu", tcbdata.TypeTask)),
		},
	}}}
}

func handleTaskCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	switch args[1] {
	case "menu":
		return editTaskMenu(ctx, update)
	case "running":
		return editRunningTasks(ctx, update)
	case "queued":
		return editQueuedTasks(ctx, update)
	case "failed":
		return editFailedTasks(ctx, update)
	default:
		return editTaskMenu(ctx, update)
	}
}

func editTaskMenu(ctx *ext.Context, update *ext.Update) error {
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     taskMenuText(),
		ReplyMarkup: taskMenuMarkup(),
	})
	return dispatcher.EndGroups
}

type taskListItem struct {
	ID      string
	Title   string
	Created time.Time
}

func taskItems(tasks []queue.TaskInfo) []taskListItem {
	items := make([]taskListItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, taskListItem{ID: task.ID, Title: task.Title, Created: task.Created})
	}
	return items
}

func taskListText(title string, total int, items []taskListItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%d)", title, total)
	limit := min(len(items), taskListLimit)
	for i := 0; i < limit; i++ {
		item := items[i]
		fmt.Fprintf(&b, "\n\n%s: %s\n%s: %s\n%s: %s",
			i18n.T(i18nk.BotMsgTasksFieldId), item.ID,
			i18n.T(i18nk.BotMsgTasksFieldTitle), item.Title,
			i18n.T(i18nk.BotMsgTasksFieldCreated), item.Created.In(time.Local).Format("2006-01-02 15:04:05"),
		)
	}
	if len(items) > limit {
		fmt.Fprintf(&b, "\n\n%s", i18n.T(i18nk.BotMsgTasksTruncatedNote, map[string]any{"Count": len(items)}))
	}
	return b.String()
}

func cancelTaskRows(items []taskListItem) []tg.KeyboardInlineButtonRow {
	rows := make([]tg.KeyboardInlineButtonRow, 0, min(len(items), taskListLimit)+1)
	for i := 0; i < min(len(items), taskListLimit); i++ {
		rows = append(rows, tg.KeyboardInlineButtonRow{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgTasksCancelButton), fmt.Appendf(nil, "%s %s", tcbdata.TypeCancel, items[i].ID)),
		}})
	}
	return rows
}

func editRunningTasks(ctx *ext.Context, update *ext.Update) error {
	tasks := core.GetRunningTasks(ctx)
	var text string
	markup := taskBackMarkup()
	if len(tasks) == 0 {
		text = i18n.T(i18nk.BotMsgTasksRunningEmpty)
	} else {
		items := taskItems(tasks)
		text = taskListText(i18n.T(i18nk.BotMsgTasksRunningTitle), len(items), items)
		markup.Rows = append(cancelTaskRows(items), markup.Rows...)
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     text,
		ReplyMarkup: markup,
	})
	return dispatcher.EndGroups
}

func editQueuedTasks(ctx *ext.Context, update *ext.Update) error {
	tasks := core.GetQueuedTasks(ctx)
	var text string
	markup := taskBackMarkup()
	if len(tasks) == 0 {
		text = i18n.T(i18nk.BotMsgTasksQueuedEmpty)
	} else {
		items := taskItems(tasks)
		text = taskListText(i18n.T(i18nk.BotMsgTasksQueuedTitle), len(items), items)
		markup.Rows = append(cancelTaskRows(items), markup.Rows...)
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     text,
		ReplyMarkup: markup,
	})
	return dispatcher.EndGroups
}

func editFailedTasks(ctx *ext.Context, update *ext.Update) error {
	failed, _, err := database.ListTaskRecords(ctx, database.TaskFilter{Status: "failed", Limit: taskListLimit})
	if err != nil {
		return err
	}
	partial, _, err := database.ListTaskRecords(ctx, database.TaskFilter{Status: "partial", Limit: taskListLimit})
	if err != nil {
		return err
	}
	records := append(failed, partial...)
	if len(records) > taskListLimit {
		records = records[:taskListLimit]
	}
	var text string
	rows := make([]tg.KeyboardInlineButtonRow, 0, len(records)+1)
	if len(records) == 0 {
		text = i18n.T(i18nk.BotMsgTasksFailedEmpty)
	} else {
		var b strings.Builder
		fmt.Fprintf(&b, "%s (%d)", i18n.T(i18nk.BotMsgTasksFailedTitle), len(records))
		for _, record := range records {
			fmt.Fprintf(&b, "\n\n%s: %s\n%s: %s\n%s: %s",
				i18n.T(i18nk.BotMsgTasksFieldId), record.TaskID,
				i18n.T(i18nk.BotMsgTasksFieldTitle), record.Title,
				i18n.T(i18nk.BotMsgTasksFieldStatus), record.Status,
			)
			rows = append(rows, tg.KeyboardInlineButtonRow{Buttons: []tg.KeyboardInlineButton{
				markup.Callback(i18n.T(i18nk.BotMsgTasksRetryButton), fmt.Appendf(nil, "%s %s", tcbdata.TypeRetry, record.TaskID)),
			}})
		}
		text = b.String()
	}
	rows = append(rows, taskBackMarkup().Rows...)
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     text,
		ReplyMarkup: &tg.ReplyInlineMarkup{Rows: rows},
	})
	return dispatcher.EndGroups
}

func handleRetryCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	taskID := args[1]
	candidates := []*ext.Context{ctx}
	if uc := userclient.GetCtx(); uc != nil {
		candidates = append(candidates, uc)
	}
	newID, count, err := retry.FailedTask(ctx, candidates, taskID)
	if err != nil {
		log.FromContext(ctx).Errorf("failed to retry task %s: %v", taskID, err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), i18n.T(i18nk.BotMsgRetryErrorFailed, map[string]any{
			"Error": err.Error(),
		})))
		return dispatcher.EndGroups
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: i18n.T(i18nk.BotMsgRetryInfoStarted, map[string]any{"TaskID": newID, "Count": count}),
	})
	return dispatcher.EndGroups
}
