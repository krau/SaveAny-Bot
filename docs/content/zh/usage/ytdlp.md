---
title: "视频下载 (yt-dlp)"
weight: 7
---

# 视频下载 (yt-dlp)

{{< hint warning >}}
需要系统安装 yt-dlp 命令行工具。
{{< /hint >}}

视频解析可通过 `/download` 使用。发送支持的视频链接后，选择 **视频（yt-dlp）**。支持 YouTube、Bilibili、Twitter 等主流音视频站点。

```bash
/download <链接1> [链接2] [参数...]
```

示例：

```bash
# 基本下载
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ

# 下载多个视频
/download https://www.youtube.com/watch?v=video1 https://www.youtube.com/watch?v=video2

# 使用自定义参数
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ -f best
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ --extract-audio --audio-format mp3
```

常用参数：

- `-f <format>`: 指定下载格式（如 `best`、`worst`、`bestvideo+bestaudio`）
- `-o <template>`: 文件名模板（如 `%(uploader)s - %(title)s.%(ext)s`）
- `--extract-audio`: 提取音频
- `--audio-format <format>`: 音频格式（如 `mp3`、`m4a`、`wav`）
- `--write-sub`: 下载字幕
- `--write-thumbnail`: 下载缩略图

出于安全考虑，以下会执行程序、加载外部配置或读写任意文件的参数会被拒绝：`--exec`、`--exec-before-download`、`--config-location`、`--load-info-json`、`--batch-file`、`--downloader`、`--external-downloader`、`--paths`、`--cache-dir`、`--postprocessor-args`、`--netrc-cmd`、`--plugin-dirs`、`--update` 等。

## 文件命名

下载文件默认按视频标题命名（`%(title)s.%(ext)s`），保存到你选择的存储目录。可以传入 yt-dlp 输出模板自定义文件名：

```bash
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ -o "%(uploader)s - %(title)s.%(ext)s"
```

只使用 `-o/--output` 的模板部分：下载目录始终由 Bot 管理，模板中的相对路径会创建在该目录下，并保留到存储路径中。如需修改所有下载的默认命名，请在 [`[ytdlp]` 配置](../deployment/configuration)中设置 `filename_template`；其中的 `restrict_filenames` 会将文件名限制为 ASCII，去掉非拉丁字符、空格和 `&`。

更多参数请参考 [yt-dlp 文档](https://github.com/yt-dlp/yt-dlp#usage-and-options).
