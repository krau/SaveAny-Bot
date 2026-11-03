---
title: "Installation and updates"
weight: 10
---

# Installation and updates

Start with an empty data directory, then configure the bot, storage, and users in the web console.

## Binary deployment

Download and extract a release for your platform:

```bash
chmod +x saveany-bot
./saveany-bot
```

The default data directory is `data/` under the current working directory. The web UI listens on `127.0.0.1:28080` by default. At startup it prints a direct URL with an access token:

```text
Web UI available at http://127.0.0.1:28080/?token=...
```

Open that URL; the token creates an authenticated session. Unless an access token is set under **Settings → Web**, a new token is generated on every start.

To expose the web UI on another interface or port, pass the listen address on the command line:

```bash
./saveany-bot --listen 0.0.0.0:28080
```

The equivalent environment variable is `SAVEANY_LISTEN`:

```bash
SAVEANY_LISTEN='0.0.0.0:28080' ./saveany-bot
```

The generated admin password remains available for password login; set `SAVEANY_ADMIN_PASSWORD` to choose it. It is printed only on the first start. After signing in, change it from the sidebar **Change password** dialog. Open the logged URL and follow the **Guide** page:

1. Enter the bot token on the Telegram page.
2. Add at least one storage on the Storages page.
3. Add your Telegram user ID on the Users page.

### Process supervision

{{< tabs "daemon" >}}
{{< tab "systemd (Linux)" >}}

Create `/etc/systemd/system/saveany-bot.service`:

```ini
[Unit]
Description=SaveAnyBot
After=systemd-user-sessions.service

[Service]
Type=simple
WorkingDirectory=/yourpath/
ExecStart=/yourpath/saveany-bot
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
systemctl enable --now saveany-bot
```

{{< /tab >}}
{{< tab "procd (OpenWrt)" >}}

Create `/etc/init.d/saveanybot`, based on [wrt_init](https://github.com/krau/SaveAny-Bot/blob/main/docs/confs/wrt_init), and adjust the working directory and executable path:

```sh
#!/bin/sh /etc/rc.common
START=99
STOP=10
description="SaveAnyBot"
WORKING_DIR="/mnt/mmc1-1/SaveAnyBot"
EXEC_PATH="$WORKING_DIR/saveany-bot"
start() { cd $WORKING_DIR && $EXEC_PATH & }
stop() { killall saveany-bot; }
```

```bash
chmod +x /etc/init.d/saveanybot
ln -s /etc/init.d/saveanybot /etc/rc.d/S99saveanybot
```

{{< /tab >}}
{{< /tabs >}}

## Docker deployment

### Docker Compose

Download [docker-compose.yml](https://github.com/krau/SaveAny-Bot/blob/main/docker-compose.yml), adjust ports and paths if needed, then start:

```bash
docker compose up -d
docker compose logs -f saveany-bot
```

Default mounts:

- `./data:/app/data` — settings, task history, and sessions
- `./downloads:/app/downloads` — default local storage
- `./cache:/app/cache` — download cache

Set an admin password, or read the generated one from the log. Set the listen address for bridge networking:

```yaml
environment:
  SAVEANY_ADMIN_PASSWORD: change-me
  SAVEANY_LISTEN: 0.0.0.0:28080
```

### Docker

```bash
docker run -d --name saveany-bot \
    -p 28080:28080 \
    -e SAVEANY_LISTEN=0.0.0.0:28080 \
    -v ./data:/app/data \
    -v ./downloads:/app/downloads \
    -v ./cache:/app/cache \
    ghcr.io/krau/saveany-bot:latest
```

Read the tokenized URL from the logs and open it. For local storage in Docker, use `/app/downloads` as the base path.

## Updates

### Binary

```bash
./saveany-bot up
```

You can also send `/update` in Telegram. The updated process restarts itself; on Windows, restart it from your service manager.

### Docker

```bash
docker pull ghcr.io/krau/saveany-bot:latest
docker restart saveany-bot
```

### Docker Compose

```bash
docker compose pull
docker compose up -d
```

Back up `data/` before updating and keep the Docker volume.
