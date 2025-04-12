package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/types"
)

func sendRuleHelp(ctx *ext.Context, update *ext.Update, userChatID int64) error {
	user, err := dao.GetUserByChatID(userChatID)
	if err != nil {
		common.Log.Errorf("获取用户规则失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户规则失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextStyledTextArray(
		[]styling.StyledTextOption{
			styling.Bold("使用方法: /rule <操作> <参数...>"),
			styling.Bold(fmt.Sprintf("\n当前已%s规则模式", map[bool]string{true: "启用", false: "禁用"}[user.ApplyRule])),
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
				for _, rule := range user.Rules {
					ruleText := fmt.Sprintf("%s %s %s %s", rule.Type, rule.Data, rule.StorageName, rule.DirPath)
					sb.WriteString(fmt.Sprintf("%d: %s\n", rule.ID, ruleText))
				}
				return sb.String()
			}(), true),
		},
	), nil)
	return dispatcher.EndGroups
}

func ruleCmd(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		return sendRuleHelp(ctx, update, update.GetUserChat().GetID())
	}
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	switch args[1] {
	case "switch":
		// /rule switch
		return switchApplyRule(ctx, update, user)
	case "add":
		// /rule add <type> <data> <storage> <dirpath>
		if len(args) < 6 {
			return sendRuleHelp(ctx, update, user.ChatID)
		}
		return addRule(ctx, update, user, args)
	case "del":
		// /rule del <id>
		if len(args) < 3 {
			return sendRuleHelp(ctx, update, user.ChatID)
		}
		ruleID := args[2]
		id, err := strconv.Atoi(ruleID)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString("无效的规则ID"), nil)
			return dispatcher.EndGroups
		}
		if err := dao.DeleteRule(uint(id)); err != nil {
			common.Log.Errorf("删除规则失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("删除规则失败"), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString("删除规则成功"), nil)
		return dispatcher.EndGroups
	default:
		return sendRuleHelp(ctx, update, user.ChatID)
	}
}

func switchApplyRule(ctx *ext.Context, update *ext.Update, user *dao.User) error {
	applyRule := !user.ApplyRule
	if err := dao.UpdateUserApplyRule(user.ChatID, applyRule); err != nil {
		common.Log.Errorf("更新用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("更新用户失败"), nil)
		return dispatcher.EndGroups
	}
	if applyRule {
		ctx.Reply(update, ext.ReplyTextString("已启用规则模式"), nil)
	} else {
		ctx.Reply(update, ext.ReplyTextString("已禁用规则模式"), nil)
	}
	return dispatcher.EndGroups
}

func addRule(ctx *ext.Context, update *ext.Update, user *dao.User, args []string) error {
	// /rule add <type> <data> <storage> <dirpath>
	ruleType := args[2]
	ruleData := args[3]
	storageName := args[4]
	dirPath := args[5]

	if !slice.Contain(types.RuleTypes, types.RuleType(ruleType)) {
		var ruleTypesStylingArray []styling.StyledTextOption
		ruleTypesStylingArray = append(ruleTypesStylingArray, styling.Bold("无效的规则类型, 可用类型:\n"))
		for i, ruleType := range types.RuleTypes {
			ruleTypesStylingArray = append(ruleTypesStylingArray, styling.Code(string(ruleType)))
			if i != len(types.RuleTypes)-1 {
				ruleTypesStylingArray = append(ruleTypesStylingArray, styling.Plain(", "))
			}
		}
		ctx.Reply(update, ext.ReplyTextStyledTextArray(ruleTypesStylingArray), nil)
		return dispatcher.EndGroups
	}
	rule := &dao.Rule{
		Type:        ruleType,
		Data:        ruleData,
		StorageName: storageName,
		DirPath:     dirPath,
		UserID:      user.ID,
	}
	if err := dao.CreateRule(rule); err != nil {
		common.Log.Errorf("添加规则失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("添加规则失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("添加规则成功"), nil)
	return dispatcher.EndGroups
}
