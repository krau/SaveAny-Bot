---
title: "Migrate from v0"
weight: 15
---

# Migrate from v0

v1 stores configuration in a local SQLite database. On first startup, it imports a legacy v0 `config.toml` once, then all settings are managed from the web console.

v1 does not migrate Telegram sessions. The v0 `session.db` and `usersession.db` files were created by gotgproto and are no longer read. Migration reuses only the business database; the bot signs in again with the configured token, and an enabled Userbot must sign in again from the web console. Back up all existing files before upgrading.

## 1. Stop and back up

Stop v0 and back up the configuration, data, and downloads:

```bash
docker compose down
cp -a config.toml data downloads /path/to/backup/
```

For a binary deployment, stop the process and back up the same directories.

Do not run v0 and v1 at the same time, and never share one `data/` directory between two instances.

## 2. Keep the data directory

v1 reuses only the business database:

| v0 file | v1 default | Purpose |
|---|---|---|
| `data/saveany.db` | `data/saveany.db` | Users, directories, rules, watches, and task history |

If v0 used a custom `db.path`, point v1 at the old database with `--db-path`, or copy the old database to `data/saveany.db`. In Docker, set `SAVEANY_DB_PATH` to the path inside the container.

The v0 `data/session.db` and `data/usersession.db` are ignored; delete them after confirming they are backed up. v1 uses new session files:

| File | Purpose |
|---|---|
| `data/bot_session.db` | Bot session |
| `data/userbot_session.db` | Userbot session |

The bot signs in again with the configured token. If the Userbot is enabled, sign in again on the Telegram page later.

## 3. Import the configuration on first startup

v1 must be able to read the legacy `config.toml` during its first startup. In Docker, mount it only for that run:

```yaml
services:
  saveany-bot:
    image: ghcr.io/krau/saveany-bot:latest
    restart: unless-stopped
    ports:
      - "28080:28080"
    environment:
      SAVEANY_LISTEN: 0.0.0.0:28080
    volumes:
      - ./data:/app/data
      - ./config.toml:/app/config.toml:ro
      - ./downloads:/app/downloads
      - ./cache:/app/cache
```

```bash
docker compose up -d
docker compose logs -f saveany-bot
```

For a binary deployment:

```bash
./saveany-bot --config /path/to/config.toml
```

Without `--config`, v1 checks `./config.toml` and then `/etc/saveany/config.toml`.

After the import log appears, remove the `config.toml` mount and restart. v1 writes an import marker to the database and will not import it again. The legacy `db.session` and `telegram.userbot.session` values are ignored; v1 always uses the new session file paths.

## 4. Sign in and verify

The startup log prints a direct Web UI URL with an access token; open it to sign in. If `SAVEANY_ADMIN_PASSWORD` was not set on the first run, the fallback admin password is printed in the log as well:

```bash
docker compose logs saveany-bot | grep -E "Web UI available at|Generated web admin password"
```

Open **Guide** after signing in and verify:

- The bot has reconnected with the current token.
- If the Userbot is enabled, it has signed in again and shows its account.
- At least one storage is enabled and has no initialization error.
- The Users list contains your old accounts. Old users that were not present in the v0 config are disabled; enable them manually if needed.
- Task history, directories, rules, and watches are still present.

Then send a file or Telegram message link to the bot and confirm a task completes.

## 5. After migration

- Later changes to `config.toml` are **ignored**. Use the web console.
- Back up the entire `data/` directory before updates and keep the Docker volume.
- Update Docker with `docker compose pull && docker compose up -d`.
- If you change the Bot token, delete `data/bot_session.db` before restarting to avoid an old session mismatch.

## Troubleshooting

**Nothing was imported**

Check that the config path is correct and the file is valid TOML. The import marker is written only after a successful import. Existing storage names and settings groups are not overwritten.

**A legacy session was not migrated**

v1 does not read the old gotgproto `session.db` or `usersession.db`. The bot signs in again with the current token; if the Userbot is enabled, sign in again on the Telegram page. Keep the old files in your backup and delete them when they are no longer needed.

**The database path is different**

v1 does not treat v0's `[db] path` as the new database location. Use `--db-path` / `SAVEANY_DB_PATH`, or move the old file.

**Local storage paths stopped working**

v0 resolved local storage paths relative to the process working directory. In Docker this is usually `/app`; make sure `downloads/` is still mounted at the same path.

**The admin password is unknown**

Find `Generated web admin password` in the first startup log. Only a hash is stored, so the password cannot be recovered from the database. If the log is gone, back up the database, delete the `auth` setting, and restart to generate a new password.

## Related

- [Installation and updates](../installation)
- [Web console configuration](../configuration/)
- [Usage](../../usage/)
