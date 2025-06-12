package handlers

import (
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
)

func handleMessageLink(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	msgLinks := re.TgMessageLinkRegexp.FindAllString(update.EffectiveMessage.GetMessage(), -1)
	for _, link := range msgLinks {
		logger.Infof("Found message link: %s", link)
	}
	panic("Not implemented")
}
