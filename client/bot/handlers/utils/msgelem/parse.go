package msgelem

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

func BuildParsedTextEntity(item parser.Item) (string, []tg.MessageEntityClass, error) {
	eb := entity.Builder{}
	if err := styling.Perform(&eb,
		styling.Bold(fmt.Sprintf("[%s]%s", item.Site, item.Title)),
		styling.Plain(i18n.T(i18nk.BotMsgParseInfoLinkPrefix, nil)),
		styling.Code(item.URL),
		styling.Plain(i18n.T(i18nk.BotMsgParseInfoAuthorPrefix, nil)),
		styling.Code(item.Author),
		styling.Plain(i18n.T(i18nk.BotMsgParseInfoDescriptionPrefix, nil)),
		styling.Blockquote(strutil.Ellipsis(item.Description, 233), true),
		styling.Plain(i18n.T(i18nk.BotMsgParseInfoFileCountPrefix, nil)),
		styling.Code(fmt.Sprintf("%d", len(item.Resources))),
		styling.Plain(i18n.T(i18nk.BotMsgParseInfoTotalSizePrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB", func() float64 {
			var totalSize int64
			for _, res := range item.Resources {
				totalSize += res.Size
			}
			return float64(totalSize) / 1024 / 1024
		}())),
		styling.Plain(i18n.T(i18nk.BotMsgParseInfoPromptSelectStorage, nil)),
	); err != nil {
		return "", nil, fmt.Errorf("failed to build parsed text entity: %w", err)
	}
	text, entities := eb.Complete()
	return text, entities, nil
}
