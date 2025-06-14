package msgelem

import "github.com/gotd/td/tg"

func AlertCallbackAnswer(queryID int64, text string) *tg.MessagesSetBotCallbackAnswerRequest {
	return &tg.MessagesSetBotCallbackAnswerRequest{
		QueryID:   queryID,
		Alert:     true,
		Message:   text,
		CacheTime: 5,
	}
}
