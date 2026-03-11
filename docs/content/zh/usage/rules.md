---
title: "存储规则"
weight: 3
---

# 存储规则

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

## FILENAME-REGEX

根据文件名正则匹配, 规则内容要求为一个合法的正则表达式, 如

```
FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /视频
```

表示将文件名后缀为 mp4,mkv,ts,avi,flv 的文件放到名为 MyAlist 存储下的 /视频 目录内 (同时受配置文件中的 `base_path` 影响)

## MESSAGE-REGEX

同上, 但是是根据消息本身的文本内容正则匹配

## IS-ALBUM

匹配相册消息 (media group), 规则内容只能为 `true` 或 `false`.

规则中的路径若使用 "NEW-FOR-ALBUM" , 则表示为该组消息新建一个文件夹来存储它们. 见: https://github.com/krau/SaveAny-Bot/issues/87

例如:

```
IS-ALBUM true MyWebdav NEW-FOR-ALBUM
```

这将会把以 media group 形式发送的消息保存到名为 MyWebdav 的存储下, 并为每个相册新建一个文件夹(由第一个文件生成)来存储它们.
