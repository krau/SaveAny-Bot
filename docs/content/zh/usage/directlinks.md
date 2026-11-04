---
title: "链接下载"
weight: 5
---

# 链接下载

使用 `/download` 命令发送一个或多个链接，并选择下载方式：

- **直接下载** — 普通 HTTP/HTTPS 文件链接
- **Aria2** — 启用 Aria2 后可用，也支持磁力和种子链接
- **视频** — 安装 yt-dlp 后可用，用于支持的视频网站

```bash
/download <链接1> [链接2] [链接3] ...
```

示例：

```bash
/download https://example.com/file.zip
/download https://example.com/file1.zip https://example.com/file2.zip
```

选择下载方式后，再选择目标存储位置。视频链接可以在同一命令后追加 yt-dlp 参数，例如 `-f best` 或 `--extract-audio`。
