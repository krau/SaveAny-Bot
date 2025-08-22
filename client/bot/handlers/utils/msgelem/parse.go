package msgelem

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

func BuildParsedTextEntity(item parser.Item) (string, []tg.MessageEntityClass, error) {
	eb := entity.Builder{}
	if err := styling.Perform(&eb,
		styling.Bold(fmt.Sprintf("[%s]%s", item.Site, item.Title)),
		styling.Plain("\n链接: "),
		styling.Code(item.URL),
		styling.Plain("\n作者: "),
		styling.Code(item.Author),
		styling.Plain("\n描述: "),
		styling.Code(strutil.Ellipsis(item.Description, 233)),
		styling.Plain("\n文件数量: "),
		styling.Code(fmt.Sprintf("%d", len(item.Resources))),
		styling.Plain("\n预计总大小: "),
		styling.Code(fmt.Sprintf("%.2f MB", func() float64 {
			var totalSize int64
			for _, res := range item.Resources {
				totalSize += res.Size
			}
			return float64(totalSize) / 1024 / 1024
		}())),
		styling.Plain("\n请选择存储位置"),
	); err != nil {
		return "", nil, fmt.Errorf("构建消息失败: %w", err)
	}
	text, entities := eb.Complete()
	return text, entities, nil
}
