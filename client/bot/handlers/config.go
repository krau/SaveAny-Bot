package handlers

import (
	"fmt"
	"strings"
	"text/template"

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

func handleConfigFnameTmpl(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	args := strings.Fields(string(update.EffectiveMessage.Text))
	if len(args) <= 1 {
		text := `使用该命令设置文件名模板, 示例:
/fnametmpl 图片_{{.msgid}}_{{.msgdate}}.jpg

可用变量:
- {{.msgid}}: 消息ID
- {{.msgtags}}: 消息中的标签, 将以下划线分隔输出
- {{.msggen}}: 根据消息生成的文件名
- {{.msgdate}}: 消息日期, 格式 YYYY-MM-DD_HH-MM-SS
- {{.origname}}: 媒体的原始文件名 (如果有)`
		if user.FilenameTemplate != "" {
			text += fmt.Sprintf("\n\n当前模板: %s", user.FilenameTemplate)
		}
		text += "\n\n模板仅在文件名策略设置为 '自定义模板' 时生效, 且模板解析错误时会回退到默认文件名"
		ctx.Reply(update, ext.ReplyTextString(text), nil)
		return dispatcher.EndGroups
	}
	newTmpl := strings.Join(args[1:], " ")
	_, err = template.New("filename").Parse(newTmpl)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("无效的模板, 请检查语法\n"+err.Error()), nil)
		return dispatcher.EndGroups
	}
	user.FilenameTemplate = newTmpl
	if err := database.UpdateUser(ctx, user); err != nil {
		return err
	}
	ctx.Reply(update, ext.ReplyTextString("已更新文件名模板"), nil)
	return dispatcher.EndGroups
}
