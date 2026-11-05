---
title: "存储规则"
weight: 3
---

# 存储规则

通过设置重定向规则，自动按文件名、消息内容或相册将保存的文件分类归档。

见: <a href="https://github.com/krau/SaveAny-Bot/issues/28" target="_blank">#28</a>

发送 `/rules` 可查看现有规则、开关规则匹配或删除规则。添加规则与导入预设命令如下：

目前支持的规则类型:

1. FILENAME-REGEX
2. MESSAGE-REGEX
3. IS-ALBUM

添加规则语法:

`/rules add <规则类型> <匹配内容> <存储ID|存储名> <路径>`

存储参数为纯数字时按存储 ID 解析（可在 Web 控制台的存储页面查看），否则按存储名解析。存储名包含空格时请使用 ID 或为名称加双引号。示例：

```
/rules add FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ 3 /视频
/rules add FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ "我的 Alist" /视频
```

若存储填为 `CHOSEN`，则表示保存到所选存储端的指定路径下。

可通过 `/rules switch` 开关规则模式。关闭时所有文件直接存入默认存储。

## 预设规则

Bot 内置了常见文件类型（视频、图片、音频、文档、压缩包）的分类规则，可一键导入：

```
/rules preset <存储ID|存储名> [基础路径]
```

参数:

- `存储`: 目标存储 ID 或名称 (必须存在且你有权访问)
- `基础路径`: 可选. 各预设分类的子目录会创建在此路径下; 若不填则直接使用默认分类目录名

示例:

```
# 导入预设规则到 "MyAlist", 使用默认目录布局
/rules preset MyAlist

# 在自定义基础路径 "downloads/sorted" 下导入预设规则
/rules preset MyAlist downloads/sorted
```

此命令会为每个分类创建 `FILENAME-REGEX` 规则, 将匹配的文件路由到 `基础路径` 下对应的子目录:

| 分类 | 匹配的扩展名 | 默认目录 |
|---|---|---|
| 视频 | mp4, mkv, ts, avi, flv, mov, webm, wmv, rmvb, m2ts | `视频` |
| 图片 | jpg, jpeg, png, gif, webp, bmp | `图片` |
| 音频 | mp3, flac, wav, aac, m4a, ogg | `音频` |
| 文档 | pdf, doc, docx, xls, xlsx, ppt, pptx, txt, md, csv, epub, mobi, azw3, chm | `文档` |
| 压缩包 | zip, rar, 7z, tar, gz, bz2, xz, ... | `压缩包` |

{{< hint info >}}
导入后的预设规则就是普通的 `FILENAME-REGEX` 规则. 你可以像其他规则一样通过 `/rules` 查看或单独删除它们.
{{< /hint >}}

规则类型:

## FILENAME-REGEX

根据文件名正则匹配, 规则内容要求为一个合法的正则表达式, 如

```
FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /视频
```

表示将文件名后缀为 mp4,mkv,ts,avi,flv 的文件放到名为 MyAlist 存储下的 /视频 目录内 (同时受存储设置中的 `base_path` 影响)

## MESSAGE-REGEX

根据消息正文或说明文字（caption）进行正则表达式匹配。

## IS-ALBUM

匹配相册消息（media group），规则内容为 `true` 或 `false`。

若路径指定为 `NEW-FOR-ALBUM`，则会为整组消息自动新建文件夹并统一保存（参考 issue [#87](https://github.com/krau/SaveAny-Bot/issues/87)）。

例如:

```
IS-ALBUM true MyWebdav NEW-FOR-ALBUM
```

这将把媒体组消息保存到 MyWebdav 存储下，并为每个相册以首个文件的名称新建子目录归档。
