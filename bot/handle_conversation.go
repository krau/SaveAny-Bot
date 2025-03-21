package bot

import (
	"sync"
)

type ConversationType string

type ConversationState struct {
	sync.Mutex
	conversationType ConversationType
	InConversation   bool
	data             map[ConversationType]map[string]interface{}
}

func (c *ConversationState) Reset() {
	c.Lock()
	defer c.Unlock()
	c.InConversation = false
	c.conversationType = ""
	c.data = make(map[ConversationType]map[string]interface{})
}

func (c *ConversationState) SetConversationType(t ConversationType) {
	c.Lock()
	defer c.Unlock()
	c.conversationType = t
}

func (c *ConversationState) GetData(key string) interface{} {
	if c.data == nil || c.data[c.conversationType] == nil {
		return nil
	}
	return c.data[c.conversationType][key]
}

func (c *ConversationState) SetData(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	if c.data == nil {
		c.data = make(map[ConversationType]map[string]interface{})
	}
	if c.data[c.conversationType] == nil {
		c.data[c.conversationType] = make(map[string]interface{})
	}
	c.data[c.conversationType][key] = value
}

// TODO: Implement conversation handling
// var userConversationState = make(map[int64]*ConversationState)

// func handleConversation(ctx *ext.Context, update *ext.Update) error {
// 	userID := update.EffectiveUser().GetID()
// 	state, ok := userConversationState[userID]
// 	if !ok {
// 		return dispatcher.ContinueGroups
// 	}
// 	if update.EffectiveMessage.Text == "/cancel" {
// 		state.Reset()
// 		ctx.Reply(update, ext.ReplyTextString("已取消"), nil)
// 		return dispatcher.EndGroups
// 	}
// 	if !state.InConversation {
// 		return dispatcher.ContinueGroups
// 	}
// 	return handleConversationState(ctx, update, state)
// }

// func handleConversationState(ctx *ext.Context, update *ext.Update, state *ConversationState) error {
// 	switch state.conversationType {
// 	default:
// 		common.Log.Errorf("Unknown conversation type: %s", state.conversationType)
// 	}
// 	return dispatcher.EndGroups
// }
