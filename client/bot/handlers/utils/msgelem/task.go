package msgelem

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

func BuildTaskAddedEntities(
	ctx context.Context,
	filename string,
	queueLength int,
) (string, []tg.MessageEntityClass) {
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := fmt.Sprintf("已添加到任务队列\n文件名: %s\n当前排队任务数: %d", filename, queueLength)
	if err := styling.Perform(&entityBuilder,
		styling.Plain("已添加到任务队列\n文件名: "),
		styling.Code(filename),
		styling.Plain("\n当前排队任务数: "),
		styling.Bold(strconv.Itoa(queueLength)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = entityBuilder.Complete()
	}
	return text, entities
}
