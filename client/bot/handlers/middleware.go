package handlers

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/config"
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
	// if err := database.CreateUser(ctx, userID); err != nil {
	// 	log.FromContext(ctx).Errorf("创建用户失败: %s", err)
	// }

	return dispatcher.ContinueGroups
}

func handleSilentSave(next func(*ext.Context, *ext.Update) error) func(*ext.Context, *ext.Update) error {
	return func(ctx *ext.Context, update *ext.Update) error {
		// TODO: implement
		return next(ctx, update)
	}
}