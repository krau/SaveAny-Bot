package msgelem

import (
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/database"
)

func BuildDirHelpStyling(user *database.User, dirs []database.Dir) []styling.StyledTextOption {
	stylings := []styling.StyledTextOption{
		styling.Bold("使用方法: /dir <操作> <参数...>"),
		styling.Plain("\n\n可用操作:\n"),
		styling.Code("add"),
		styling.Plain(" <存储名> <路径> - 添加路径\n"),
		styling.Code("del"),
		styling.Plain(" <路径ID> - 删除路径\n"),
		styling.Code("setdefault"),
		styling.Plain(" <路径ID> - 设置默认路径\n"),
		styling.Code("cleardefault"),
		styling.Plain(" - 清除默认路径\n"),
		styling.Plain("\n添加路径示例:\n"),
		styling.Code("/dir add local1 path/to/dir"),
		styling.Plain("\n\n删除路径示例:\n"),
		styling.Code("/dir del 3"),
		styling.Plain("\n\n设置默认路径示例:\n"),
		styling.Code("/dir setdefault 3"),
	}
	
	if user.DefaultDirID != nil {
		stylings = append(stylings, styling.Plain("\n\n当前默认路径ID: "))
		stylings = append(stylings, styling.Bold(fmt.Sprintf("%d", *user.DefaultDirID)))
	}
	
	stylings = append(stylings, styling.Plain("\n\n当前已添加的路径:\n"))
	stylings = append(stylings, styling.Blockquote(func() string {
		var sb strings.Builder
		for _, dir := range dirs {
			sb.WriteString(fmt.Sprintf("%d: ", dir.ID))
			sb.WriteString(dir.StorageName)
			sb.WriteString(" - ")
			sb.WriteString(dir.Path)
			if user.DefaultDirID != nil && *user.DefaultDirID == dir.ID {
				sb.WriteString(" [默认]")
			}
			sb.WriteString("\n")
		}
		return sb.String()
	}(), true))
	
	return stylings
}
