package msgelem

const (
	SaveHelpText = `
	使用方法:

	1. 使用该命令回复要保存的文件, 可选文件名参数.
	示例:
	/save custom_file_name.mp4

	2. 设置默认存储后, 发送 /save <频道ID/用户名> <消息ID范围> 来批量保存文件. 遵从存储规则, 若未匹配到任何规则则使用默认存储.
	示例:
	/save @acherkrau 114-514
	`
)
