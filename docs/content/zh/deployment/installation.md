---
title: "安装与更新"
weight: 10
---

# 安装与更新

准备空数据目录即可启动，随后在 Web 控制台完成 Bot、存储和用户配置。

## 从预编译文件部署

在 [Release](https://github.com/krau/SaveAny-Bot/releases) 页面下载对应平台的二进制文件并解压：

```bash
chmod +x saveany-bot
./saveany-bot
```

程序默认使用当前目录下的 `data/` 保存数据库和会话。Web UI 默认监听 `127.0.0.1:28080`，启动时会输出带访问 Token 的直达地址：

```text
Web UI available at http://127.0.0.1:28080/?token=...
```

打开该地址即可登录。未在“设置 → Web”中指定访问 Token 时，每次启动均会重新生成。

如需在其他网卡或端口访问，通过命令行指定监听地址：

```bash
./saveany-bot --listen 0.0.0.0:28080
```

对应的环境变量为 `SAVEANY_LISTEN`：

```bash
SAVEANY_LISTEN='0.0.0.0:28080' ./saveany-bot
```

自动生成的管理员密码用于首次登录，也可通过环境变量 `SAVEANY_ADMIN_PASSWORD` 预设。初始密码仅在首次启动时打印，登录后可在侧栏修改密码。打开日志中的直达地址后，参照指南页面完成初始化：

1. 在 Telegram 页面填写 Bot Token。
2. 在存储页面添加至少一个存储。
3. 在用户页面添加你的 Telegram User ID。

### 进程守护

{{< tabs "daemon" >}}
{{< tab "systemd (常规 Linux)" >}}

创建 `/etc/systemd/system/saveany-bot.service`：

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

创建 `/etc/init.d/saveanybot`，参考 [wrt_init](https://github.com/krau/SaveAny-Bot/blob/main/docs/confs/wrt_init) 并修改工作目录与程序路径：

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

## 使用 Docker 部署

### Docker Compose

下载 [docker-compose.yml](https://github.com/krau/SaveAny-Bot/blob/main/docker-compose.yml)，按需修改端口和数据目录，然后启动：

```bash
docker compose up -d
docker compose logs -f saveany-bot
```

默认挂载：

- `./data:/app/data` — 设置、任务记录和会话
- `./downloads:/app/downloads` — 本地存储的默认目录
- `./cache:/app/cache` — 下载缓存

可指定管理员密码；未指定时从首启日志读取生成的密码。使用桥接网络时需将监听地址设为 0.0.0.0：

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
    ghcr.io/krau/saveany-bot:v1-beta
```

打开日志中输出的直达链接。Docker 部署时，链接中的 `127.0.0.1` 指宿主机；若从其他设备访问，请替换为宿主机 IP。若使用本地存储，容器内路径可设为 `/app/downloads`。

`v1-beta` 是持续指向最新 v1 Beta 的移动 tag；需要可复现部署时，请改用具体的 beta 版本 tag。

## 更新

### 预编译二进制

```bash
./saveany-bot up
```

或在 Telegram 中发送 `/update`。更新后进程会自动重启（Windows 环境需手动或由服务管理器重启）。

### Docker

```bash
docker pull ghcr.io/krau/saveany-bot:v1-beta
docker restart saveany-bot
```

### Docker Compose

```bash
docker compose pull
docker compose up -d
```

更新前请备份 `data/`，并保留 Docker 数据卷。
