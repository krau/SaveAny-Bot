---
title: "CLI Subcommands"
weight: 21
---

# CLI Subcommands

Besides running the Telegram bot with `./saveany-bot` (no subcommand), the binary exposes two helper subcommands for moving local files into a storage backend: `upload` (one-shot) and `watch` (continuous).

These subcommands load the same `config.toml` as the bot, initialize the database and caches, then perform their task. They do **not** start the Telegram bot itself, although storages of type `telegram` will spin up the bot client just for the upload.

## `upload` — Upload a Single File

```
saveany-bot upload -f <file> -s <storage> [-d <dir>] [--no-progress]
```

Flags:

| Flag | Required | Description |
|---|---|---|
| `-f, --file` | Yes | Path to the local file to upload |
| `-s, --storage` | Yes | Target storage name (must exist in `config.toml`) |
| `-d, --dir` | No | Destination directory within the storage. Defaults to the storage's `base_path` |
| `--no-progress` | No | Disable the terminal progress bar |

Examples:

```bash
# Upload a file to the default dir of storage "MyAlist"
./saveany-bot upload -f ./movie.mp4 -s MyAlist

# Upload into a specific subdirectory
./saveany-bot upload -f ./movie.mp4 -s MyAlist -d movies/2026

# Upload via Telegram storage without a progress bar
./saveany-bot upload -f ./photo.jpg -s MyChannel --no-progress
```

## `watch` — Watch a Directory and Auto-Upload

The `watch` subcommand continuously monitors a local directory and uploads created or modified files to a storage backend, preserving the relative directory structure from the watch root.

```
saveany-bot watch -p <path> -s <storage> [-d <dir>] [options]
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `-p, --path` | *(required)* | Local directory to watch |
| `-s, --storage` | *(required)* | Target storage name |
| `-d, --dir` | storage's `base_path` | Destination directory within the storage |
| `-r, --recursive` | `false` | Watch subdirectories recursively |
| `--overwrite` | `false` | Overwrite existing files on the storage instead of skipping them |
| `--initial-scan` | `false` | Upload files already present in the directory on startup |
| `--debounce` | `2s` | How long to wait after the last write before uploading a file |
| `--upload-workers` | `config.workers` | Number of concurrent uploads |
| `--retry-delay` | `3s` | Delay between upload retries |

{{< hint info >}}
Write-completion detection: the watcher debounces per file and only uploads once the file size stays unchanged across the debounce window, so partial/write-in-progress files are not uploaded.
<br />
If a file changes while being uploaded, it is re-uploaded once after the current upload finishes (instead of being queued multiple times).
{{< /hint >}}

Examples:

```bash
# Watch ./inbox and upload new files to "MyAlist" recursively
./saveany-bot watch -p ./inbox -s MyAlist -r

# Watch with a custom destination dir and overwrite
./saveany-bot watch -p ./inbox -s MyAlist -d backup --overwrite

# On startup, also upload everything already in ./inbox
./saveany-bot watch -p ./inbox -s MyAlist --initial-scan
```

### Behavior notes

- Relative directory structure is preserved under the destination directory. A file written to `./inbox/sub/file.txt` with `--path ./inbox` is uploaded to `<dest_dir>/sub/file.txt`.
- `watch` runs until interrupted (e.g. `Ctrl-C` / `SIGINT`); in-flight uploads are drained before exit.
- Retries follow the global `retry` value from `config.toml`, with `--retry-delay` between attempts.
- Telegram-type storages will start the bot client automatically to perform uploads.

{{< hint warning >}}
`watch` is unrelated to the in-bot `/watch` command (which watches Telegram chats). This subcommand watches a **local filesystem directory** and uploads to a storage backend, independent of Telegram.
{{< /hint >}}