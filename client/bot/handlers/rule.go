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
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/rule"
)

func handleRuleCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strutil.ParseArgsRespectQuotes(update.EffectiveMessage.Text)
	userChatID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user rules: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorGetUserRulesFailed, nil)), nil)
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
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorUpdateUserFailed, nil)), nil)
			return dispatcher.EndGroups
		}
		if applyRule {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleInfoRuleModeEnabled, nil)), nil)
		} else {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleInfoRuleModeDisabled, nil)), nil)
		}
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
			return rule.RuleType(""), fmt.Errorf("invalid rule type: %s\navailable: %v", ruleTypeArg, slice.Join(rule.Values(), ", "))
		}()
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorInvalidRuleType, map[string]any{
				"Type":      ruleTypeArg,
				"Available": slice.Join(rule.Values(), ", "),
			})), nil)
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
			logger.Errorf("failed to create rule: %s", err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorCreateRuleFailed, nil)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleInfoCreateRuleSuccess, nil)), nil)
	case "del":
		// /rule del <id>
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRulePromptProvideRuleId, nil)), nil)
			return dispatcher.EndGroups
		}
		ruleID := args[2]
		id, err := strconv.Atoi(ruleID)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorInvalidRuleId, nil)), nil)
			return dispatcher.EndGroups
		}
		if err := database.DeleteRule(ctx, uint(id)); err != nil {
			logger.Errorf("failed to delete rule %d: %s", id, err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorDeleteRuleFailed, nil)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleInfoDeleteRuleSuccess, nil)), nil)
	default:
		ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildRuleHelpStyling(user.ApplyRule, user.Rules)), nil)
		return dispatcher.EndGroups
	}
	return dispatcher.EndGroups
}
