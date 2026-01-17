package msgelem

import (
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/message/styling"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
)

func BuildDirHelpStyling(dirs []database.Dir) []styling.StyledTextOption {
	return []styling.StyledTextOption{
		styling.Bold(i18n.T(i18nk.BotMsgDirHelpUsage, nil)),
		styling.Plain(i18n.T(i18nk.BotMsgDirHelpAvailableOps, nil)),
		styling.Code("add"),
		styling.Plain(i18n.T(i18nk.BotMsgDirHelpAddSuffix, nil)),
		styling.Code("del"),
		styling.Plain(i18n.T(i18nk.BotMsgDirHelpDelSuffix, nil)),
		styling.Plain(i18n.T(i18nk.BotMsgDirHelpAddExamplePrefix, nil)),
		styling.Code(i18n.T(i18nk.BotMsgDirHelpAddExampleCmd, nil)),
		styling.Plain(i18n.T(i18nk.BotMsgDirHelpDelExamplePrefix, nil)),
		styling.Code(i18n.T(i18nk.BotMsgDirHelpDelExampleCmd, nil)),
		styling.Plain(i18n.T(i18nk.BotMsgDirHelpExistingDirsPrefix, nil)),
		styling.Blockquote(func() string {
			var sb strings.Builder
			for _, dir := range dirs {
				fmt.Fprintf(&sb, "%d: ", dir.ID)
				sb.WriteString(dir.StorageName)
				sb.WriteString(" - ")
				sb.WriteString(dir.Path)
				sb.WriteString("\n")
			}
			return sb.String()
		}(), true),
	}
}
