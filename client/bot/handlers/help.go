package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/pkg/consts"
)

func handleHelpCmd(ctx *ext.Context, update *ext.Update) error {
	const helpText string = `
Save Any Bot - 转存你的 Telegram 文件
版本: %s , 提交: %s
命令:
/start - 开始使用
/help - 显示帮助
/silent - 开关静默模式
/storage - 设置默认存储位置
/save [自定义文件名] - 保存文件

静默模式: 开启后 Bot 直接保存到收到的文件到默认位置, 不再询问

默认存储位置: 在静默模式下保存到的位置

向 Bot 发送(转发)文件, 或发送一个公开频道的消息链接以保存文件
`
	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf(helpText, consts.Version, consts.GitCommit)), nil)
	return dispatcher.EndGroups
}
