---
title: "Configuration"
weight: 11
---

# Configuration

Settings are stored in the SQLite database under the data directory (default `data/saveany.db`) and managed from the web console. The first start uses defaults and prints a direct web URL with an access token; set `SAVEANY_ADMIN_PASSWORD` to choose the fallback password.

## Configuration pages

| Page | Contents |
|---|---|
| Guide | First-run checklist and everyday usage |
| Telegram | Bot token, API ID/hash, proxy, Userbot login |
| Storages | Local disk, WebDAV, Alist, S3, MinIO, Rclone, Telegram, and more |
| Users | Telegram user IDs, default storage, directories, rules, and access |
| Settings | Task concurrency, language, proxy, Aria2, yt-dlp, parsers, cache, logging, web, and API |

Some settings require a process restart, including task concurrency, global proxy, database session path, cache settings, plugins, hooks, and the web listen address. The page marks fields that need a restart.

### Telegram

Create a bot with [@BotFather](https://t.me/BotFather). `API ID` and `API Hash` have defaults and usually need no changes; use your own [Telegram API](https://my.telegram.org/apps) credentials when enabling the Userbot.

Saving bot settings starts the connection automatically. Userbot login happens on the Telegram page, with no terminal interaction inside the container.

### Helper bots

Helper bots are optional. Add them in the **Helper bots** section of the **Telegram** page with a BotFather token, then add each bot to the destination Telegram chat with permission to send files. Telegram storage uploads are distributed across running helpers that can access the destination; helpers without access are skipped and uploads fall back to the primary bot. Helper bots do not handle commands.

To split large file downloads across helpers, set a private **Relay channel ID** on the **Telegram** page, make sure the primary bot can post there, and add at least two helper bots to that channel. The primary bot forwards incoming files to the relay channel, and each helper obtains its own file reference before downloading assigned chunks. This requires non-stream mode and files of at least 8 MB, and falls back to the single-client downloader when the relay or helper pool is unavailable. Forwarded relay messages are deleted after download by default; enable **Keep forwarded channel messages** to retain them.

### Storages

Add and enable storages on the **Storages** page. Local disk needs only a base path:

- In Docker, usually `/app/downloads`.
- For a binary deployment, use `./downloads` or an absolute path.

See [Storage configuration](./storages) for backend fields.

### Users

Only enabled Telegram user IDs on the **Users** page can use the bot. A new user needs at least the numeric user ID; the default storage can be set later.

## Environment variables

| Variable | Purpose |
|---|---|
| `SAVEANY_DATA_DIR` | Data directory, default `./data` |
| `SAVEANY_DB_PATH` | SQLite file path, default `<data-dir>/saveany.db` |
| `SAVEANY_LISTEN` | Web listen address, default `127.0.0.1:28080`; use `0.0.0.0:28080` to expose it |
| `SAVEANY_ADMIN_PASSWORD` | Set the fallback admin password on first start; otherwise one is generated and logged |

## Security

- The web console binds to `127.0.0.1:28080` by default. Expose it publicly only through a reverse proxy with HTTPS.
- The `Access token` setting enables direct Web UI access through `/?token=...` and Bearer authentication for `/api/v1`. When empty, a token is generated on every start and printed in the log.
- Change the generated fallback admin password immediately after the first password login.
- Never expose `data/`, bot tokens, Userbot sessions, or access tokens.
