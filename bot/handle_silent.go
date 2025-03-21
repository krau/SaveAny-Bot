package bot

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
)

func silent(ctx *ext.Context, update *ext.Update) error {
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		return dispatcher.EndGroups
	}
	if !user.Silent && user.DefaultStorage == "" {
		ctx.Reply(update, ext.ReplyTextString("请先使用 /storage 设置默认存储位置"), nil)
		return dispatcher.EndGroups
	}
	user.Silent = !user.Silent
	if err := dao.UpdateUser(user); err != nil {
		common.Log.Errorf("更新用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("更新用户失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("已%s静默模式", map[bool]string{true: "开启", false: "关闭"}[user.Silent])), nil)
	return dispatcher.EndGroups
}
