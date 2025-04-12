package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
)

func sendDirHelp(ctx *ext.Context, update *ext.Update, userChatID int64) error {
	dirs, err := dao.GetUserDirsByChatID(userChatID)
	if err != nil {
		common.Log.Errorf("获取用户路径失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户路径失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextStyledTextArray(
		[]styling.StyledTextOption{
			styling.Bold("使用方法: /dir <操作> <参数...>"),
			styling.Plain("\n\n可用操作:\n"),
			styling.Code("add"),
			styling.Plain(" <存储名> <路径> - 添加路径\n"),
			styling.Code("del"),
			styling.Plain(" <路径ID> - 删除路径\n"),
			styling.Plain("\n添加路径示例:\n"),
			styling.Code("/dir add local1 path/to/dir"),
			styling.Plain("\n\n删除路径示例:\n"),
			styling.Code("/dir del 3"),
			styling.Plain("\n\n当前已添加的路径:\n"),
			styling.Blockquote(func() string {
				var sb strings.Builder
				for _, dir := range dirs {
					sb.WriteString(fmt.Sprintf("%d: ", dir.ID))
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

func dirCmd(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		return sendDirHelp(ctx, update, update.GetUserChat().GetID())
	}
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	switch args[1] {
	case "add":
		// /dir add local1 path/to/dir
		if len(args) < 4 {
			return sendDirHelp(ctx, update, update.GetUserChat().GetID())
		}
		return addDir(ctx, update, user, args[2], args[3])
	case "del":
		// /dir del 3
		if len(args) < 3 {
			return sendDirHelp(ctx, update, update.GetUserChat().GetID())
		}
		dirID, err := strconv.Atoi(args[2])
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString("路径ID无效"), nil)
			return dispatcher.EndGroups
		}
		return delDir(ctx, update, user, dirID)
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

func delDir(ctx *ext.Context, update *ext.Update, user *dao.User, dirID int) error {
	if err := dao.DeleteDirByID(uint(dirID)); err != nil {
		common.Log.Errorf("删除路径失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("删除路径失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("路径删除成功"), nil)
	return dispatcher.EndGroups
}
