package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/config"
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
/dir - 管理存储目录
/rule - 管理规则
/update - 检查更新并升级

使用帮助: https://sabot.unv.app/usage
反馈群组: https://t.me/ProjectSaveAny
`
	shortHash := config.GitCommit
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf(helpText, config.Version, shortHash)), nil)
	return dispatcher.EndGroups
}
