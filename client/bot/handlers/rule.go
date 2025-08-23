package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/rule"
)

func handleRuleCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	userChatID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("获取用户规则失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户规则失败"), nil)
		return dispatcher.EndGroups
	}
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildRuleHelpStyling(user.ApplyRule, user.Rules)), nil)
		return dispatcher.EndGroups
	}
	switch args[1] {
	case "switch":
		// /rule switch
		applyRule := !user.ApplyRule
		if err := database.UpdateUserApplyRule(ctx, user.ChatID, applyRule); err != nil {
			logger.Errorf("更新用户失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("更新用户失败"), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("已%s规则模式", map[bool]string{true: "启用", false: "禁用"}[applyRule])), nil)
	case "add":
		// /rule add <type> <data> <storage> <dirpath>
		if len(args) < 6 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildRuleHelpStyling(user.ApplyRule, user.Rules)), nil)
			return dispatcher.EndGroups
		}
		ruleTypeArg := args[2]
		ruleType, err := func() (rule.RuleType, error) {
			for _, t := range rule.Values() {
				if strings.EqualFold(t.String(), ruleTypeArg) {
					return t, nil
				}
			}
			return rule.RuleType(""), fmt.Errorf("无效的规则类型: %s\n可用: %v", ruleTypeArg, slice.Join(rule.Values(), ", "))
		}()
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(err.Error()), nil)
			return dispatcher.EndGroups
		}

		ruleData := args[3]
		storageName := args[4]
		dirPath := args[5]

		rd := &database.Rule{
			Type:        ruleType.String(),
			Data:        ruleData,
			StorageName: storageName,
			DirPath:     dirPath,
			UserID:      user.ID,
		}
		if err := database.CreateRule(ctx, rd); err != nil {
			logger.Errorf("创建规则失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("创建规则失败"), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString("创建规则成功"), nil)
	case "del":
		// /rule del <id>
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextString("请提供规则ID"), nil)
			return dispatcher.EndGroups
		}
		ruleID := args[2]
		id, err := strconv.Atoi(ruleID)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString("无效的规则ID"), nil)
			return dispatcher.EndGroups
		}
		if err := database.DeleteRule(ctx, uint(id)); err != nil {
			logger.Errorf("删除规则失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("删除规则失败"), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString("删除规则成功"), nil)
	default:
		ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildRuleHelpStyling(user.ApplyRule, user.Rules)), nil)
		return dispatcher.EndGroups
	}
	return dispatcher.EndGroups
}
