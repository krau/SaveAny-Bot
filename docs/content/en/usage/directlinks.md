---
title: "URL Downloads"
weight: 5
---

# URL Downloads

Use the `/download` command with one or more links and select an engine:

- **Direct** — plain HTTP/HTTPS download, best for ordinary file links
- **Aria2** — when Aria2 is enabled, including magnet and torrent links
- **Video** — when yt-dlp is available, for supported video sites

```bash
/download <url1> [url2] [url3] ...
```

Examples:

```bash
/download https://example.com/file.zip
/download https://example.com/file1.zip https://example.com/file2.zip
```

Choose the engine, then select the target storage. For video sites, yt-dlp options such as `-f best` or `--extract-audio` can be appended to the same command.
