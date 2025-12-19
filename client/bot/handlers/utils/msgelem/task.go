package msgelem

import (
	"context"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
)

func BuildTaskAddedEntities(
	ctx context.Context,
	filename string,
	queueLength int,
) (string, []tg.MessageEntityClass) {
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := i18n.T(i18nk.BotMsgTasksInfoAddedToQueueFull, map[string]any{
		"Filename":    filename,
		"QueueLength": queueLength,
	})
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgTasksInfoAddedToQueuePrefix, nil)),
		styling.Plain(i18n.T(i18nk.BotMsgTasksInfoFilenamePrefix, nil)),
		styling.Code(filename),
		styling.Plain(i18n.T(i18nk.BotMsgTasksInfoQueueLengthPrefix, nil)),
		styling.Bold(strconv.Itoa(queueLength)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = entityBuilder.Complete()
	}
	return text, entities
}
