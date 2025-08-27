package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
)

func handleHelpCmd(ctx *ext.Context, update *ext.Update) error {
	shortHash := config.GitCommit
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf(i18n.T(i18nk.BotMsgHelpTextFmt), config.Version, shortHash)), nil)
	return dispatcher.EndGroups
}
