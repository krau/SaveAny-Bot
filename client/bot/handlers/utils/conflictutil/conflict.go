package conflictutil

import (
	"fmt"
	"strings"

	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
)

const maxConflictLines = 10

func EffectiveStrategy(user *database.User) string {
	if user != nil && tcbdata.IsConflictStrategy(user.ConflictStrategy) {
		return user.ConflictStrategy
	}
	return tcbdata.ConflictStrategyRename
}

func ResolveStrategy(user *database.User, override string) string {
	if tcbdata.IsConflictStrategy(override) {
		return override
	}
	return EffectiveStrategy(user)
}

func Display(strategy string) string {
	switch strategy {
	case tcbdata.ConflictStrategyRename:
		return i18n.T(i18nk.BotMsgConfigConflictStrategyRename, nil)
	case tcbdata.ConflictStrategyAsk:
		return i18n.T(i18nk.BotMsgConfigConflictStrategyAsk, nil)
	case tcbdata.ConflictStrategyOverwrite:
		return i18n.T(i18nk.BotMsgConfigConflictStrategyOverwrite, nil)
	case tcbdata.ConflictStrategySkip:
		return i18n.T(i18nk.BotMsgConfigConflictStrategySkip, nil)
	default:
		return strategy
	}
}

func FormatPaths(conflicts []string) string {
	if len(conflicts) <= maxConflictLines {
		return strings.Join(conflicts, "\n")
	}
	return strings.Join(conflicts[:maxConflictLines], "\n") + "\n" + i18n.T(i18nk.BotMsgCommonPromptConflictMoreFiles, map[string]any{
		"Count": len(conflicts) - maxConflictLines,
	})
}

func FormatPath(storageName, storagePath string) string {
	return fmt.Sprintf("[%s]:%s", storageName, storagePath)
}
