package handlers

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
)

func handleConfigCmd(ctx *ext.Context, update *ext.Update) error {
	ctx.Reply(update, ext.ReplyTextString("请选择要配置的选项"), &ext.ReplyOpts{
		Markup: &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "文件名策略",
							Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeConfig, "fnamest"),
						},
					},
				},
			},
		},
	})
	return dispatcher.EndGroups
}

func handleConfigCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	invaildDataAnswer := func() error {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Alert:     true,
			Message:   "无效的回调数据",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	if len(args) < 2 {
		return invaildDataAnswer()
	}
	switch args[1] {
	case "fnamest":
		return handleConfigFnameSTCallback(ctx, update)
	default:
		return invaildDataAnswer()
	}
}

func handleConfigFnameSTCallback(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) == 3 {
		selected := args[2]
		st, err := fnamest.ParseFnameST(selected)
		if err != nil {
			return err
		}
		user.FilenameStrategy = st.String()
		if err := database.UpdateUser(ctx, user); err != nil {
			return err
		}
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      update.CallbackQuery.GetMsgID(),
			Message: fmt.Sprintf("已将文件名策略设置为: %s", fnamest.FnameSTDisplay[st]),
		})
		return dispatcher.EndGroups
	}
	opts := fnamest.FnameSTValues()
	buttons := make([]tg.KeyboardButtonClass, 0, len(opts))
	for _, opt := range opts {
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: fnamest.FnameSTDisplay[opt],
			Data: fmt.Appendf(nil, "%s %s %s", tcbdata.TypeConfig, "fnamest", opt),
		})
	}
	markup := &tg.ReplyInlineMarkup{Rows: []tg.KeyboardButtonRow{
		{Buttons: buttons},
	}}
	currentStStr := user.FilenameStrategy
	if currentStStr == "" {
		currentStStr = fnamest.Default.String()
	}
	currentSt, err := fnamest.ParseFnameST(currentStStr)
	if err != nil {
		currentSt = fnamest.Default
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     fmt.Sprintf("请选择文件名策略, 当前策略: %s", fnamest.FnameSTDisplay[currentSt]),
		ReplyMarkup: markup,
	})
	return dispatcher.EndGroups
}
