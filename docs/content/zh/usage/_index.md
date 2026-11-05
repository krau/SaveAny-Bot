---
title: "使用帮助"
weight: 10
---

# 使用帮助

介绍 Save Any Bot 的常用功能与命令操作。如需部署及高级配置，请参阅[配置说明](../deployment/configuration)；其他问题可在 GitHub [Discussions](https://github.com/krau/SaveAny-Bot/discussions) 交流。

## Bot 命令

| 命令 | 用途 |
|---|---|
| `/start` | 打开帮助和命令列表 |
| `/save` | 保存回复的媒体消息，或按消息范围批量保存 |
| `/download` | 使用直接下载、Aria2 或视频解析下载链接 |
| `/transfer` | 在存储端之间传输文件 |
| `/tasks` | 查看和管理运行中、排队中和失败任务 |
| `/config` | 用户设置：文件名和重名策略、静默模式、默认存储 |
| `/template` | 设置自定义文件名模板 |
| `/dirs` | 管理存储目录 |
| `/rules` | 管理存储规则 |
| `/watch` | 监听聊天并管理通知/对话同步 (UserBot) |
| `/parser` | 管理解析器插件 |
| `/update` | 检查更新 |

## 转存文件

向 Bot 发送或转发以下内容即可触发转存：

1. 图片、视频、音频、文档等媒体文件
2. Telegram 消息链接（如 `https://t.me/acherkrau/1097`），支持转存开启了防转发/防保存保护的频道文件
3. Telegra.ph 文章链接，自动提取并下载其中的所有图片

## 在群组中使用

Bot 支持普通群组和超级群组，不支持广播频道。

- 群组中仅响应显式命令和按钮回调，不会自动保存群内的普通媒体或普通文本消息。
- 转存群内文件时，回复目标消息并发送 `/save`（群组中亦可使用 `/save@YourBot` 格式）；其他命令（如 `/download`）用法相同。
- 仅白名单用户可以使用。非白名单成员的消息会被忽略，点击内联按钮将提示无权限。
- 超级群中以匿名管理员身份发送的消息因无法识别具体用户，会被直接忽略。
- 默认的 Privacy Mode 下，Bot 仅能接收针对它的命令、@提及以及对其消息的回复。若需接收群内全部消息，可在 [@BotFather](https://t.me/BotFather) 使用 `/setprivacy` 关闭 Privacy Mode；Bot 在群组中依然仅响应命令和交互按钮。