<div align="center">

# <img src="docs/logo.jpg" width="45" align="center"> Save Any Bot

[简体中文](README.md) | **English**

Save Telegram files to various storage endpoints.

> _Just like PikPak Bot_

</div>

## Deployment

### Deploy from Binary

Download the binary file for your platform from the [Release](https://github.com/krau/SaveAny-Bot/releases) page.

Create a `config.toml` file in the extracted directory, refer to [config.example.toml](https://github.com/krau/SaveAny-Bot/blob/main/config.example.toml) for configuration.

Run:

```bash
chmod +x saveany-bot
./saveany-bot
```

#### Add as systemd Service

Create file `/etc/systemd/system/saveany-bot.service` and write the following content:

```
[Unit]
Description=SaveAnyBot
After=systemd-user-sessions.service

[Service]
Type=simple
WorkingDirectory=/yourpath/
ExecStart=/yourpath/saveany-bot
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Enable auto-start and start the service:

```bash
systemctl enable --now saveany-bot
```

### Deploy with Docker

#### Docker Compose

Download [docker-compose.yml](https://github.com/krau/SaveAny-Bot/blob/main/docker-compose.yml) file and create a `config.toml` file in the same directory, refer to [config.example.toml](https://github.com/krau/SaveAny-Bot/blob/main/config.example.toml) for configuration.

Run:

```bash
docker compose up -d
```

#### Docker

```shell
docker run -d --name saveany-bot \
    -v /path/to/config.toml:/app/config.toml \
    -v /path/to/downloads:/app/downloads \
    ghcr.io/krau/saveany-bot:latest
```

## Update

Use `upgrade` or `up` command to upgrade to the latest version:

```bash
./saveany-bot upgrade
```

If deployed with Docker, use the following commands to update:

```bash
docker pull ghcr.io/krau/saveany-bot:latest
docker restart saveany-bot
```

## Usage

Send (forward) files to the Bot and follow the prompts.

---

## Thanks

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- All the dependencies