package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/rule"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

// resolveRuleStorage accepts a storage ID or a storage name. IDs avoid the
// ambiguity of names that contain spaces; CHOSEN keeps the rule storage-less.
func resolveRuleStorage(userChatID int64, arg string) (uint, error) {
	if strings.EqualFold(arg, rule.RuleStorNameChosen) {
		return 0, nil
	}
	if id, err := strconv.ParseUint(arg, 10, 32); err == nil {
		if !config.C().HasStorageID(userChatID, uint(id)) {
			return 0, fmt.Errorf("storage ID %d not found", id)
		}
		return uint(id), nil
	}
	if !config.C().HasStorage(userChatID, arg) {
		return 0, fmt.Errorf("storage not found: %s", arg)
	}
	return config.C().GetStorageIDByName(arg), nil
}

func handleRuleCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strutil.ParseArgsRespectQuotes(update.EffectiveMessage.Text)
	userChatID := responsibleUserID(update)
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user rules: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorGetUserRulesFailed, nil)), nil)
		return dispatcher.EndGroups
	}
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(ruleMenuText(user)), &ext.ReplyOpts{Markup: ruleMenuMarkup(user)})
		return dispatcher.EndGroups
	}
	switch args[1] {
	case "switch":
		// /rules switch
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
		// /rules add <type> <data> <storage> <dirpath>
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
		storageArg := args[4]
		dirPath := args[5]
		storageID, err := resolveRuleStorage(user.ChatID, storageArg)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorStorageNotFound, map[string]any{
				"Storage": storageArg,
			})), nil)
			return dispatcher.EndGroups
		}

		rd := &database.Rule{
			Type:      ruleType.String(),
			Content:   ruleData,
			StorageID: storageID,
			DirPath:   dirPath,
			UserID:    user.ID,
		}
		if err := database.CreateRule(ctx, rd); err != nil {
			logger.Errorf("failed to create rule: %s", err)
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorCreateRuleFailed, nil)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleInfoCreateRuleSuccess, nil)), nil)
	case "preset":
		// /rules preset <storage> [base_path]
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildRuleHelpStyling(user.ApplyRule, user.Rules)), nil)
			return dispatcher.EndGroups
		}
		storageArg := args[2]
		storageID, err := resolveRuleStorage(user.ChatID, storageArg)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorStorageNotFound, map[string]any{
				"Storage": storageArg,
			})), nil)
			return dispatcher.EndGroups
		}
		basePath := ""
		if len(args) >= 4 {
			basePath = args[3]
		}
		presets := rule.PresetCategories(basePath)
		imported := 0
		for _, p := range presets {
			rd := &database.Rule{
				Type:      rule.FileNameRegex.String(),
				Content:   p.Regex,
				StorageID: storageID,
				DirPath:   p.Dir,
				UserID:    user.ID,
			}
			if err := database.CreateRule(ctx, rd); err != nil {
				logger.Errorf("failed to create preset rule %s: %s", p.Name, err)
				continue
			}
			imported++
		}
		if imported == 0 {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleErrorCreateRuleFailed, nil)), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgRuleInfoPresetImported, map[string]any{
			"Count": imported,
		})), nil)
	case "del":
		// /rules del <id>
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
		if err := database.DeleteRule(ctx, uint(id), user.ID); err != nil {
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

func ruleMenuText(user *database.User) string {
	var b strings.Builder
	b.WriteString(i18n.T(i18nk.BotMsgRuleMenuTitle))
	b.WriteString("\n\n")
	b.WriteString(i18n.T(i18nk.BotMsgRuleMenuHint))
	mode := i18nk.BotMsgRuleInfoRuleModeDisabled
	if user.ApplyRule {
		mode = i18nk.BotMsgRuleInfoRuleModeEnabled
	}
	b.WriteString("\n\n")
	b.WriteString(i18n.T(mode))
	if len(user.Rules) == 0 {
		b.WriteString("\n\n")
		b.WriteString(i18n.T(i18nk.BotMsgRuleMenuEmpty))
		return b.String()
	}
	for _, r := range user.Rules {
		storageName := rule.RuleStorNameChosen
		if r.StorageID != 0 {
			storageName = config.C().GetStorageNameByID(r.StorageID)
			if storageName == "" {
				storageName = fmt.Sprintf("#%d", r.StorageID)
			}
		}
		fmt.Fprintf(&b, "\n\n#%d %s %s %s %s", r.ID, r.Type, r.Content, storageName, r.DirPath)
	}
	return b.String()
}

func ruleMenuMarkup(user *database.User) *tg.ReplyInlineMarkup {
	rows := make([]tg.KeyboardInlineButtonRow, 0, len(user.Rules)+1)
	label := i18nk.BotMsgRuleButtonEnable
	if user.ApplyRule {
		label = i18nk.BotMsgRuleButtonDisable
	}
	rows = append(rows, tg.KeyboardInlineButtonRow{Buttons: []tg.KeyboardInlineButton{
		markup.Callback(i18n.T(label), fmt.Appendf(nil, "%s switch", tcbdata.TypeRule)),
	}})
	for _, rule := range user.Rules {
		rows = append(rows, tg.KeyboardInlineButtonRow{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgRuleButtonDelete), fmt.Appendf(nil, "%s del %d", tcbdata.TypeRule, rule.ID)),
		}})
	}
	return &tg.ReplyInlineMarkup{Rows: rows}
}

func handleRuleCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	switch args[1] {
	case "switch":
		if err := database.UpdateUserApplyRule(ctx, user.ChatID, !user.ApplyRule); err != nil {
			return err
		}
	case "del":
		if len(args) < 3 {
			return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
		}
		ruleID, err := strconv.Atoi(args[2])
		if err != nil {
			return err
		}
		if err := database.DeleteRule(ctx, uint(ruleID), user.ID); err != nil {
			return err
		}
	}
	user, err = database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     ruleMenuText(user),
		ReplyMarkup: ruleMenuMarkup(user),
	})
	return dispatcher.EndGroups
}
