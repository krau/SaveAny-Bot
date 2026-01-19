---
title: "Usage"
weight: 10
---

# Usage

This page introduces some of Save Any Bot's features and basic usage. If you can't find what you need here, please also see the [Configuration Guide](../deployment/configuration) or ask in GitHub [Discussions](https://github.com/krau/SaveAny-Bot/discussions).

## File Transfer

To use the bot's Telegram file saving feature, you need to send or forward the following types of messages to the bot:

1. File or media messages, such as images, videos, documents, etc.
2. Telegram message links, for example: `https://t.me/acherkrau/1097`. **Even if the channel prohibits forwarding and saving, the bot can still download its files.**
3. Telegra.ph article links. The bot will download all images in the article.

## Silent Mode (silent)

Use the `/silent` command to toggle silent mode.

By default, silent mode is off, and the bot will ask you for the save location of each file.

When silent mode is enabled, the bot will save files directly to the default location without confirmation.

Before enabling silent mode, you need to set the default save location using the `/storage` command.


## Storage Rules

Storage rules allow you to define redirection rules when the bot uploads files to storage, so that saved files are automatically organized.

See: <a href="https://github.com/krau/SaveAny-Bot/issues/28" target="_blank">#28</a>

Currently supported rule types:

1. FILENAME-REGEX
2. MESSAGE-REGEX
3. IS-ALBUM

Basic syntax for adding rules:

"RuleType RuleContent StorageName Path"

Pay attention to spaces; the bot can only parse correctly formatted syntax. Below is an example of a valid rule command:

```
/rule add FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /videos
```

In addition, if `CHOSEN` is used as the storage name in the rule, it means files will be stored under the path of the storage you selected by clicking the inline button.

Rule types:

### FILENAME-REGEX

Matches based on filename regex. The rule content must be a valid regular expression, such as:

```
FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /videos
```

This means files with extensions mp4, mkv, ts, avi, flv will be saved to the `/videos` directory in the storage named `MyAlist` (also affected by the `base_path` in the configuration file).

### MESSAGE-REGEX

Similar to the above, but matches based on the text content of the message itself.

### IS-ALBUM

Matches album messages (media groups). Rule content can only be `true` or `false`.

If the path in the rule uses `NEW-FOR-ALBUM`, the bot will create a new folder for each media group and store all files of that group there. See: https://github.com/krau/SaveAny-Bot/issues/87

For example:

```
IS-ALBUM true MyWebdav NEW-FOR-ALBUM
```

This will save media-group messages to the storage named `MyWebdav`, creating a new folder (generated from the first file) for each album.

## Watch Chats

{{< hint warning >}}
This feature requires enabling UserBot integration.
{{< /hint >}}

You can watch messages in a specific chat and automatically save them to the default storage, following storage rules. You can also add filters so that only matching messages are saved.

Watch a chat:

```
/watch <chat_id/username> [filter]
```

Stop watching:

```
/unwatch <chat_id/username>
```

Filter types:

### msgre

Regex-match the message text. For example:

```
/watch 12345678 msgre:.*hello.*
```

This will watch the chat with ID `12345678`, and only save messages whose text contains `hello`.

## Direct Download Links

Use the `/dl` command to directly download one or more HTTP/HTTPS files to storage.

```bash
/dl <url1> [url2] [url3] ...
```

Examples:

```bash
/dl https://example.com/file.zip
/dl https://example.com/file1.zip https://example.com/file2.zip
```

The bot will validate the link format and then ask you to select the target storage location.

## Aria2 Download

{{< hint warning >}}
This feature requires enabling Aria2 in the configuration file and configuring the RPC connection.
{{< /hint >}}

Use the `/aria2dl` command to download files via the Aria2 download manager, supporting HTTP/HTTPS, FTP, BitTorrent, and other protocols.

```bash
/aria2dl <uri1> [uri2] [uri3] ...
```

Examples:

```bash
# Download HTTP link
/aria2dl https://example.com/file.zip

# Download magnet link
/aria2dl magnet:?xt=urn:btih:...

# Download torrent file (need to upload .torrent file first)
/aria2dl https://example.com/file.torrent
```

Configure Aria2:

Add to `config.toml`:

```toml
[aria2]
enable = true
url = "http://localhost:6800/jsonrpc"
secret = "your-rpc-secret"  # If rpc-secret is configured
remove_after_transfer = true  # Remove local files after transfer
```

## yt-dlp Video Download

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
- `--extract-audio`: Extract audio
- `--audio-format <format>`: Audio format (e.g., `mp3`, `m4a`, `wav`)
- `--write-sub`: Download subtitles
- `--write-thumbnail`: Download thumbnail

For more parameters, see [yt-dlp documentation](https://github.com/yt-dlp/yt-dlp#usage-and-options).

## Storage Transfer

Use the `/transfer` command to transfer files directly between different storages without going through Telegram.

```bash
/transfer <source_storage>:/<source_path> [filter]
```

Parameters:

- `source_storage`: Source storage name
- `source_path`: Source path
- `filter`: Optional regex filter to transfer only matching files

Examples:

```bash
# Transfer entire directory
/transfer local1:/downloads

# Transfer files from specified path
/transfer alist1:/media/photos

# Transfer only mp4 files
/transfer webdav1:/videos ".*\.mp4$"

# Transfer image files
/transfer local1:/pictures "(?i)\.(jpg|png|gif)$"
```

The bot will:

1. List all files in the source path
2. Apply the filter (if provided)
3. Display file count and total size
4. Ask you to select the target storage
5. Ask you to select the target directory (if configured for that storage)
6. Start the transfer task

Notes:

- Source storage must support listing and reading
- Target storage must support writing
- Real-time progress is displayed during transfer
- Transfer tasks can be cancelled

## Save Files Outside Telegram

Besides files on Telegram, the bot can also save files from other websites via JavaScript plugins or built-in parsers.

> See the [Contributing Parsers](../contribute) document for details.

Just send links that match the requirements of a parser to the bot. Currently built-in parsers include:

- Twitter
- Kemono