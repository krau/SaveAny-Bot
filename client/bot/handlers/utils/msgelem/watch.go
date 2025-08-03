package msgelem

const (
	WatchHelpText = `
使用 /watch 命令监听一个聊天的消息, 并自动保存到默认存储中, 遵从存储规则.

命令语法:
/watch <chat_id> [filter]

参数:
- <chat_id>: 聊天的 ID 或用户名
- [filter]: 可选, 格式为 过滤器类型:表达式 , 所有支持类型的过滤器请查看文档

命令示例:
/watch 2229835658 msgre:.*plana.*

这将监听 ID 为 2229835658 的聊天, 并转存所有包含 "plana" 的媒体消息
	`
)
