package msgelem

import (
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
)

func BuildRuleHelpStyling(enabled bool, rules []database.Rule) []styling.StyledTextOption {
	return []styling.StyledTextOption{
		styling.Bold(i18n.T(i18nk.BotMsgRuleHelpUsage, nil)),
		styling.Bold(func() string {
			if enabled {
				return i18n.T(i18nk.BotMsgRuleHelpCurrentModeEnabled, nil)
			}
			return i18n.T(i18nk.BotMsgRuleHelpCurrentModeDisabled, nil)
		}()),
		styling.Plain(i18n.T(i18nk.BotMsgRuleHelpAvailableOps, nil)),
		styling.Code("switch"),
		styling.Plain(i18n.T(i18nk.BotMsgRuleHelpSwitchSuffix, nil)),
		styling.Code("add"),
		styling.Plain(i18n.T(i18nk.BotMsgRuleHelpAddSuffix, nil)),
		styling.Code("del"),
		styling.Plain(i18n.T(i18nk.BotMsgRuleHelpDelSuffix, nil)),
		styling.Plain(i18n.T(i18nk.BotMsgRuleHelpExistingRulesPrefix, nil)),
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
