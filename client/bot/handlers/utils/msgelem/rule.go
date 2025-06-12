package msgelem

import (
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/database"
)

func BuildRuleHelpStyling(enabled bool, rules []database.Rule) []styling.StyledTextOption {
	return []styling.StyledTextOption{
		styling.Bold("使用方法: /rule <操作> <参数...>"),
		styling.Bold(fmt.Sprintf("\n当前已%s规则模式", map[bool]string{true: "启用", false: "禁用"}[enabled])),
		styling.Plain("\n\n可用操作:\n"),
		styling.Code("switch"),
		styling.Plain(" - 开关规则模式\n"),
		styling.Code("add"),
		styling.Plain(" <类型> <数据> <存储名> <路径> - 添加规则\n"),
		styling.Code("del"),
		styling.Plain(" <规则ID> - 删除规则\n"),
		styling.Plain("\n当前已添加的规则:\n"),
		styling.Blockquote(func() string {
			var sb strings.Builder
			for _, rule := range rules {
				ruleText := fmt.Sprintf("%s %s %s %s", rule.Type, rule.Data, rule.StorageName, rule.DirPath)
				sb.WriteString(fmt.Sprintf("%d: %s\n", rule.ID, ruleText))
			}
			return sb.String()
		}(), true),
	}
}
