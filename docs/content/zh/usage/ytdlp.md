---
title: "yt-dlp 视频下载"
weight: 7
---

# yt-dlp 视频下载

{{< hint warning >}}
该功能需要在系统中安装 yt-dlp 命令行工具.
{{< /hint >}}

使用 `/ytdlp` 命令可以下载支持的视频网站的视频和音频, 支持 YouTube、Bilibili、Twitter 等 1000+ 个网站.

```bash
/ytdlp <url1> [url2] [flags...]
```

示例:

```bash
# 基本下载
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ

# 下载多个视频
/ytdlp https://www.youtube.com/watch?v=video1 https://www.youtube.com/watch?v=video2

# 使用自定义参数
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ -f best
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ --extract-audio --audio-format mp3
```

常用参数:

- `-f <format>`: 指定下载格式 (如 `best`, `worst`, `bestvideo+bestaudio`)
- `-o <template>`: 文件名模板 (如 `%(uploader)s - %(title)s.%(ext)s`)
- `--extract-audio`: 提取音频
- `--audio-format <format>`: 音频格式 (如 `mp3`, `m4a`, `wav`)
- `--write-sub`: 下载字幕
- `--write-thumbnail`: 下载缩略图

## 文件名

下载的文件以视频标题命名 (`%(title)s.%(ext)s`), 并保存到你所选择的存储目录中. 可以通过 yt-dlp 的 output template 自定义文件名:

```bash
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ -o "%(uploader)s - %(title)s.%(ext)s"
```

`-o/--output` 只有模板部分会被使用: 下载目录始终由 bot 管理, 模板中的相对路径会创建在该目录下. 如需修改所有下载的默认命名, 请在 [`[ytdlp]` 配置](../deployment/configuration)中设置 `filename_template` (以及 `restrict_filenames`, 开启后会丢弃非拉丁字符).

更多参数请参考 [yt-dlp 文档](https://github.com/yt-dlp/yt-dlp#usage-and-options).
