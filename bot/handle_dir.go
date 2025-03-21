package bot

import (
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
)

func dirCmd(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(strings.TrimPrefix(update.EffectiveMessage.Text, "/dir "), " ")
	if len(args) < 3 {
		dirs, err := dao.GetUserDirsByChatID(update.GetUserChat().GetID())
		if err != nil {
			common.Log.Errorf("获取用户路径失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("获取用户路径失败"), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextStyledTextArray(
			[]styling.StyledTextOption{
				styling.Bold("使用方法: /dir <操作> <存储名> <路径>"),
				styling.Plain("\n\n可用操作:\n"),
				styling.Code("add"),
				styling.Plain(" - 添加路径\n"),
				styling.Code("del"),
				styling.Plain(" - 删除路径\n"),
				styling.Plain("\n示例:\n"),
				styling.Code("/dir add local1 path/to/dir"),
				styling.Plain("\n\n当前已添加的路径:\n"),
				styling.Blockquote(func() string {
					var sb strings.Builder
					for _, dir := range dirs {
						sb.WriteString(dir.StorageName)
						sb.WriteString(" - ")
						sb.WriteString(dir.Path)
						sb.WriteString("\n")
					}
					return sb.String()
				}(), true),
			},
		), nil)
		return dispatcher.EndGroups
	}
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	switch args[0] {
	case "add":
		return addDir(ctx, update, user, args[1], args[2])
	case "del":
		return delDir(ctx, update, user, args[1], args[2])
	default:
		ctx.Reply(update, ext.ReplyTextString("未知操作"), nil)
		return dispatcher.EndGroups
	}
}

func addDir(ctx *ext.Context, update *ext.Update, user *dao.User, storageName, path string) error {
	if _, err := storage.GetStorageByUserIDAndName(user.ChatID, storageName); err != nil {
		ctx.Reply(update, ext.ReplyTextString(err.Error()), nil)
		return dispatcher.EndGroups
	}

	if err := dao.CreateDirForUser(user.ID, storageName, path); err != nil {
		common.Log.Errorf("创建路径失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("创建路径失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("路径添加成功"), nil)
	return dispatcher.EndGroups
}

func delDir(ctx *ext.Context, update *ext.Update, user *dao.User, storageName, path string) error {
	if err := dao.DeleteDirForUser(user.ID, storageName, path); err != nil {
		common.Log.Errorf("删除路径失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("删除路径失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("路径删除成功"), nil)
	return dispatcher.EndGroups
}
