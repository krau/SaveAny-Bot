---
title: "配置说明"
weight: 11
---

# 配置说明

设置保存在数据目录下的 SQLite 数据库（默认 `data/saveany.db`），通过 Web 控制台管理。首次启动使用默认值，管理员密码会打印在启动日志中，也可以用 `SAVEANY_ADMIN_PASSWORD` 指定。

## 配置页面

| 页面 | 内容 |
|---|---|
| 指南 | 首次配置检查清单和日常使用说明 |
| Telegram | Bot Token、API ID/Hash、代理、Userbot 登录 |
| 存储 | 本地磁盘、WebDAV、Alist、S3、MinIO、Rclone、Telegram 等存储端 |
| 用户 | Telegram 用户 ID、默认存储、目录、规则和访问控制 |
| 设置 | 并发、语言、代理、Aria2、yt-dlp、解析器、缓存、日志、Web 和 API |

部分设置需要重启进程后生效，例如 Worker 数量、全局代理、数据库会话路径、缓存、插件、Hook 和 Web 监听地址。保存时页面会提示需要重启的字段。

### Telegram

Bot Token 在 [@BotFather](https://t.me/BotFather) 创建。`API ID` 和 `API Hash` 已提供默认值，通常无需修改；启用 Userbot 时建议使用自己的 [Telegram API](https://my.telegram.org/apps) 凭据。

保存 Bot 配置后会自动启动连接。Userbot 的登录在 Telegram 页面完成，不需要进入容器终端。

### 存储

存储端在 **存储** 页面添加并启用。本地磁盘只需要一个基础路径：

- Docker 部署通常填写 `/app/downloads`。
- 二进制部署可以填写 `./downloads` 或绝对路径。

存储端字段说明见 [存储端配置](./storages)。

### 用户

只有 **用户** 页面中启用的 Telegram User ID 能使用 Bot。新建用户时至少填写数字 User ID；默认存储可以留空，之后再设置。

## 环境变量

| 变量 | 说明 |
|---|---|
| `SAVEANY_DATA_DIR` | 数据目录，默认 `./data` |
| `SAVEANY_DB_PATH` | SQLite 文件路径，默认 `<data-dir>/saveany.db` |
| `SAVEANY_LISTEN` | Web 监听地址，例如 `0.0.0.0:8080` |
| `SAVEANY_ADMIN_PASSWORD` | 首次启动时设置管理员密码；未设置则自动生成并打印 |

## 安全

- Web 控制台和 HTTP API 共用监听端口；若暴露到公网，请使用反向代理和 HTTPS。
- 设置中的 `API Token` 用于 Bearer Token 调用 `/api/v1` 接口；为空时 API 依赖登录会话。
- 首次登录后立即修改自动生成的管理员密码。
- 不要公开 `data/`、Bot Token、Userbot 会话或 API Token。
