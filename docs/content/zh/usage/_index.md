---
title: "使用帮助"
weight: 10
---

# 使用帮助

这里介绍 Save Any Bot 的一些功能和使用方法, 如果你没有在这里找到你需要的内容, 另请参阅 [配置说明](../deployment/configuration) 或前往 Github [Discussions](https://github.com/krau/SaveAny-Bot/discussions) 提问.

## 转存文件

要使用 Bot 的转存 Telegram 文件功能, 需要向 Bot 发送或转发以下类型的消息.

1. 文件或媒体消息, 如图片, 视频, 文档等
2. Telegram 消息链接, 例如: `https://t.me/acherkrau/1097`. **即使频道禁止了转发和保存, Bot 依然可以下载其文件.**
3. Telegra.ph 的文章链接, Bot 将下载其中的所有图片

## 在群组中使用

Bot 支持普通群组和超级群组, 不支持广播频道.

- 群组中只响应显式命令和按钮回调, 不会自动保存群里的普通媒体或链接消息.
- 转存群里的文件时, 请回复目标消息并发送 `/save`, 群组中也可以使用 `/save@YourBot` 形式; 其他命令 (如 `/dl`, `/ytdlp`) 用法不变.
- 只有白名单用户可以使用. 其他成员的消息会被静默忽略, 点击按钮会收到无权限提示.
- 超级群中匿名管理员发送的消息没有可识别的用户, 会被忽略.
- 默认的 Privacy Mode 下, Bot 只能收到发给它的命令、@ 提及和对它消息的回复. 如需让 Bot 接收群内所有消息, 请在 [@BotFather](https://t.me/BotFather) 使用 `/setprivacy` 关闭 Privacy Mode, 并将 Bot 重新加入群组; 本 Bot 在群组中仍只响应命令和按钮.