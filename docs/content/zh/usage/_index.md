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

## 静默模式 (silent)

使用 `/silent` 命令可以开关静默模式.

默认情况下不开启静默模式, Bot 会询问你每个文件的保存位置.

开启静默模式后, Bot 会直接保存文件到默认位置, 无需确认.

在开启静默模式之前, 需要使用 `/storage` 命令设置默认保存位置.

## 存储规则

允许你为 Bot 在上传文件到存储时设置一些重定向规则, 用于自动整理所保存的文件.

见: <a href="https://github.com/krau/SaveAny-Bot/issues/28" target="_blank">#28</a>

目前支持的规则类型:

1. FILENAME-REGEX
2. MESSAGE-REGEX
3. IS-ALBUM

添加规则的基本语法:

"规则类型 规则内容 存储名 路径"

注意空格的使用, 语法正确 bot 才能解析, 以下是一条合法的添加规则命令:

```
/rule add FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /视频
```

此外, 规则中的存储名若使用 "CHOSEN" , 则表示存储到点击按钮选择的存储端的路径下

规则类型:

### FILENAME-REGEX

根据文件名正则匹配, 规则内容要求为一个合法的正则表达式, 如

```
FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /视频
```

表示将文件名后缀为 mp4,mkv,ts,avi,flv 的文件放到名为 MyAlist 存储下的 /视频 目录内 (同时受配置文件中的 `base_path` 影响)

### MESSAGE-REGEX

同上, 但是是根据消息本身的文本内容正则匹配

### IS-ALBUM

匹配相册消息 (media group), 规则内容只能为 `true` 或 `false`.

规则中的路径若使用 "NEW-FOR-ALBUM" , 则表示为该组消息新建一个文件夹来存储它们. 见: https://github.com/krau/SaveAny-Bot/issues/87

例如:

```
IS-ALBUM true MyWebdav NEW-FOR-ALBUM
```

这将会把以 media group 形式发送的消息保存到名为 MyWebdav 的存储下, 并为每个相册新建一个文件夹(由第一个文件生成)来存储它们.


## 监听聊天

{{< hint warning >}}
该功能需开启 UserBot 集成.
{{< /hint >}}

监听指定聊天的消息, 并自动保存到默认存储中, 遵从存储规则, 并且可以设置过滤器来只保存匹配的消息.

监听聊天:

```
/watch <chat_id/username> [filter] 
```

取消监听:

```
/unwatch <chat_id/username>
```

过滤器类型:

### msgre

正则匹配消息文本, 例如:

```
/watch 12345678 msgre:.*hello.*
```

这将会监听 ID 为 12345678 的聊天, 并且只保存消息文本中包含 "hello" 的消息.

## 转存 Telegram 之外的文件

除了 Telegram 上的文件, Bot 还可通过 JavaScript 插件或内置解析器来支持转存其他网站的文件.

> 查看[贡献解析器](../contribute)文档了解详情

只需向 Bot 发送符合解析器要求的链接即可使用, 当前内置的解析器:

- Twitter
- Kemono