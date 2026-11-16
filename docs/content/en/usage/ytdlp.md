---
title: "Video Downloads (yt-dlp)"
weight: 7
---

# Video Downloads (yt-dlp)

{{< hint warning >}}
This feature requires the yt-dlp command-line tool installed on your system.
{{< /hint >}}

Video extraction is available through `/download`. Send a supported video URL, then choose **Video (yt-dlp)**. It supports YouTube, Bilibili, Twitter, and 1000+ other sites.

```bash
/download <url1> [url2] [flags...]
```

Examples:

```bash
# Basic download
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ

# Download multiple videos
/download https://www.youtube.com/watch?v=video1 https://www.youtube.com/watch?v=video2

# Use custom parameters
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ -f best
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ --extract-audio --audio-format mp3
```

Common parameters:

- `-f <format>`: Specify download format (e.g., `best`, `worst`, `bestvideo+bestaudio`)
- `-o <template>`: File name template (e.g., `%(uploader)s - %(title)s.%(ext)s`)
- `--extract-audio`: Extract audio
- `--audio-format <format>`: Audio format (e.g., `mp3`, `m4a`, `wav`)
- `--write-sub`: Download subtitles
- `--write-thumbnail`: Download thumbnail

For safety, flags that execute programs, load external configuration, or read and write arbitrary files are rejected: `--exec`, `--exec-before-download`, `--config-location`, `--load-info-json`, `--batch-file`, `--downloader`, `--external-downloader`, `--paths`, `--cache-dir`, `--postprocessor-args`, `--netrc-cmd`, `--plugin-dirs`, `--update` and similar. `--proxy`, `--cookies` and `--netrc` are rejected too; set proxies in the settings instead (see below).

## File names

Downloaded files are named after the video title (`%(title)s.%(ext)s`) and saved into the storage directory you selected. Pass a yt-dlp output template to customize the name:

```bash
/download https://www.youtube.com/watch?v=dQw4w9WgXcQ -o "%(uploader)s - %(title)s.%(ext)s"
```

Only the template part of `-o/--output` is used: the download directory is always managed by the bot, and any relative path in the template is created below it and kept in the storage path. To change the default for every download, set `filename_template` in the [`[ytdlp]` config](../deployment/configuration); `restrict_filenames` there keeps file names ASCII-only, which drops non-latin characters, spaces and `&`.

## Proxies

Set **Settings → yt-dlp → yt-dlp proxies** to route yt-dlp through HTTP, HTTPS or SOCKS proxies (`http`, `https`, `socks4`, `socks4a`, `socks5`, `socks5h`), one per line. They are separate from the Telegram proxy and the global proxy, which do not reach the yt-dlp process. In a v0 `config.toml`, the same list is `[ytdlp] proxies = [...]` and is imported on first start.

yt-dlp accepts only one proxy, so the bot rotates for it: proxies are tried in the order listed, each with a full yt-dlp run, and the first that succeeds is used. Blank and duplicate lines are ignored; with no proxies, yt-dlp connects directly. The same list applies to downloads with custom flags and to the media metadata API.

A URL that fails everywhere, such as a deleted video, costs one run per proxy before the task fails, so list only proxies you expect to work. Cancelling a task stops the rotation. Passwords in proxy URLs are hidden in logs and task errors.

For more parameters, see [yt-dlp documentation](https://github.com/yt-dlp/yt-dlp#usage-and-options).
