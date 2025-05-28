# 实验性功能

这里的功能不太稳定, 且未来可能会被删除或修改。

## 存储规则

允许你为 Bot 在上传文件到存储时设置一些重定向规则, 用于自动整理所保存的文件.

见: https://github.com/krau/SaveAny-Bot/issues/28

目前支持的规则类型:

1. FILENAME-REGEX
2. MESSAGE-REGEX

添加规则的基本语法:

"规则类型 规则内容 存储名 路径"

注意空格的使用, 语法正确 bot 才能解析, 以下是一条合法的添加规则命令:

```
/rule add FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /视频
```

此外, 规则中的存储名若使用 "CHOSEN" , 则表示存储到点击按钮选择的存储端的路径下

规则介绍:

### FILENAME-REGEX

根据文件名正则匹配, 规则内容要求为一个合法的正则表达式, 如

```
FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /视频
```

表示将文件名后缀为 mp4,mkv,ts,avi,flv 的文件放到名为 MyAlist 存储下的 /视频 目录内 (同时受配置文件中的 `base_path` 影响)

### MESSAGE-REGEX

同上, 根据消息文本内容正则匹配

## 复制并发送媒体消息

将接收到的文件(媒体)消息, 或链接对应的消息原样发送到当前聊天, 点击选择存储按钮中的 "发送到当前聊天" 即可.
