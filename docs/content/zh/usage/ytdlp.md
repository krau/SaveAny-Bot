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
- `--extract-audio`: 提取音频
- `--audio-format <format>`: 音频格式 (如 `mp3`, `m4a`, `wav`)
- `--write-sub`: 下载字幕
- `--write-thumbnail`: 下载缩略图

更多参数请参考 [yt-dlp 文档](https://github.com/yt-dlp/yt-dlp#usage-and-options).
