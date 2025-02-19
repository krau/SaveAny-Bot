<div align="center">

# <img src="docs/logo.jpg" width="45" align="center"> Save Any Bot

**简体中文** | [English](README_EN.md) 

把 Telegram 的文件保存到各类存储端.

> _就像 PikPak Bot 一样_

</div

Demo Video:

<div align="center">

[SaveAny-Bot 演示视频 ｜ The Demo of SaveAny-Bot.webm](https://github.com/user-attachments/assets/a0de2453-a4d1-4a12-81fb-9d84856dce09)

</div>

## 部署

### 从二进制文件部署

在 [Release](https://github.com/krau/SaveAny-Bot/releases) 页面下载对应平台的二进制文件.

在解压后目录新建 `config.toml` 文件, 参考 [config.toml.example](https://github.com/krau/SaveAny-Bot/blob/main/config.example.toml) 编辑配置文件.

运行:

```bash
chmod +x saveany-bot
./saveany-bot
```

#### 添加为 systemd 服务

创建文件 `/etc/systemd/system/saveany-bot.service` 并写入以下内容:

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

设为开机启动并启动服务:

```bash
systemctl enable --now saveany-bot
```

### 使用 Docker 部署

#### Docker Compose

下载 [docker-compose.yml](https://github.com/krau/SaveAny-Bot/blob/main/docker-compose.yml) 文件, 在同目录下新建 `config.toml` 文件, 参考 [config.toml.example](https://github.com/krau/SaveAny-Bot/blob/main/config.example.toml) 编辑配置文件.

启动:

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

## 更新

使用 `upgrade` 或 `up` 升级到最新版

```bash
./saveany-bot upgrade
```

如果是 Docker 部署, 使用以下命令更新:

```bash
docker pull ghcr.io/krau/saveany-bot:latest
docker restart saveany-bot
```

## 使用

向 Bot 发送(转发)文件, 或发送公开频道的消息链接, 按照提示操作.

---

## Thanks

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- All the dependencies
