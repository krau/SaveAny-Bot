---
title: "yt-dlp Video Download"
weight: 7
---

# yt-dlp Video Download

{{< hint warning >}}
This feature requires the yt-dlp command-line tool installed on your system.
{{< /hint >}}

Use the `/ytdlp` command to download videos and audio from supported video websites, including YouTube, Bilibili, Twitter, and 1000+ other sites.

```bash
/ytdlp <url1> [url2] [flags...]
```

Examples:

```bash
# Basic download
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ

# Download multiple videos
/ytdlp https://www.youtube.com/watch?v=video1 https://www.youtube.com/watch?v=video2

# Use custom parameters
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ -f best
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ --extract-audio --audio-format mp3
```

Common parameters:

- `-f <format>`: Specify download format (e.g., `best`, `worst`, `bestvideo+bestaudio`)
- `-o <template>`: File name template (e.g., `%(uploader)s - %(title)s.%(ext)s`)
- `--extract-audio`: Extract audio
- `--audio-format <format>`: Audio format (e.g., `mp3`, `m4a`, `wav`)
- `--write-sub`: Download subtitles
- `--write-thumbnail`: Download thumbnail

## File names

Downloaded files are named after the video title (`%(title)s.%(ext)s`) and saved into the storage directory you selected. Pass a yt-dlp output template to customize the name:

```bash
/ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ -o "%(uploader)s - %(title)s.%(ext)s"
```

Only the template part of `-o/--output` is used: the download directory is always managed by the bot, and any relative path in the template is created below it and kept in the storage path. To change the default for every download, set `filename_template` in the [`[ytdlp]` config](../deployment/configuration); `restrict_filenames` there keeps file names ASCII-only, which drops non-latin characters, spaces and `&`.

For more parameters, see [yt-dlp documentation](https://github.com/yt-dlp/yt-dlp#usage-and-options).
