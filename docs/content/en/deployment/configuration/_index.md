---
title: "Configuration"
weight: 11
---

# Configuration

Settings are stored in the SQLite database under the data directory (default `data/saveany.db`) and managed from the web console. The first start uses defaults; the admin password is printed in the startup log, or set it with `SAVEANY_ADMIN_PASSWORD`.

## Configuration pages

| Page | Contents |
|---|---|
| Guide | First-run checklist and everyday usage |
| Telegram | Bot token, API ID/hash, proxy, Userbot login |
| Storages | Local disk, WebDAV, Alist, S3, MinIO, Rclone, Telegram, and more |
| Users | Telegram user IDs, default storage, directories, rules, and access |
| Settings | Concurrency, language, proxy, Aria2, yt-dlp, parsers, cache, logging, web, and API |

Some settings require a process restart, including worker count, global proxy, database session path, cache settings, plugins, hooks, and the web listen address. The page marks fields that need a restart.

### Telegram

Create a bot with [@BotFather](https://t.me/BotFather). `API ID` and `API Hash` have defaults and usually need no changes; use your own [Telegram API](https://my.telegram.org/apps) credentials when enabling the Userbot.

Saving bot settings starts the connection automatically. Userbot login happens on the Telegram page, with no terminal interaction inside the container.

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
| `SAVEANY_LISTEN` | Web listen address, for example `0.0.0.0:8080` |
| `SAVEANY_ADMIN_PASSWORD` | Set the admin password on first start; otherwise one is generated and logged |

## Security

- The web console and HTTP API share the listener. Use a reverse proxy and HTTPS if exposed publicly.
- The `API Token` setting enables Bearer authentication for `/api/v1`; when empty, the API relies on the signed-in session.
- Change the generated admin password immediately after the first login.
- Never expose `data/`, bot tokens, Userbot sessions, or API tokens.
