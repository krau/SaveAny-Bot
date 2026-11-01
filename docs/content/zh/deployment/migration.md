---
title: "从 v0 升级"
weight: 15
---

# 从 v0 升级

v1 使用本地 SQLite 数据库保存配置。首次启动时，如果检测到 v0 的 `config.toml`，会把它导入数据库一次；之后所有配置都在 Web 控制台管理。

迁移不会删除现有数据库、Bot 会话或存储文件。升级前仍建议完整备份。

## 1. 停止并备份

先停止 v0，再备份配置、数据和下载目录：

```bash
docker compose down
cp -a config.toml data downloads /path/to/backup/
```

如果使用二进制部署，直接停止进程后备份相同目录即可。

不要同时运行 v0 和 v1，也不要让两个实例共用同一个 `data/` 目录。

## 2. 保留数据目录

v1 默认复用以下文件：

| v0 文件 | v1 默认路径 | 说明 |
|---|---|---|
| `data/saveany.db` | `data/saveany.db` | 用户、目录、规则、监听和任务记录 |
| `data/session.db` | `data/session.db` | Bot 会话 |
| `data/usersession.db` | `data/usersession.db` | Userbot 会话 |

如果 v0 使用了自定义的 `db.path`，用 `--db-path` 指向旧数据库，或把旧数据库文件复制为 `data/saveany.db`。Docker 部署可通过 `SAVEANY_DB_PATH` 指定容器内路径。

## 3. 首次启动时导入配置

首次启动必须让 v1 能读到旧版 `config.toml`。Docker 只在这时挂载配置文件：

```yaml
services:
  saveany-bot:
    image: ghcr.io/krau/saveany-bot:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
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

二进制部署：

```bash
./saveany-bot --config /path/to/config.toml
```

省略 `--config` 时，v1 会依次查找工作目录下的 `./config.toml` 和 `/etc/saveany/config.toml`。

日志中出现配置导入完成后，即可移除 `config.toml` 挂载并重启。v1 在数据库中写入导入标记，不会重复导入。

## 4. 登录并核对

首次启动如果没有设置 `SAVEANY_ADMIN_PASSWORD`，管理员密码会打印在日志中：

```bash
docker compose logs saveany-bot | grep "admin password"
```

登录后打开 **指南**，确认：

- Bot Token 和连接状态正确。
- 存储已启用且没有初始化错误。
- 用户列表包含原有账号。v0 配置中不存在的旧用户会被禁用，需要时手动启用。
- 任务记录、目录、规则和监听仍然存在。

然后向 Bot 发送一个文件或 Telegram 消息链接，确认任务能正常完成。

## 5. 迁移之后

- `config.toml` 的后续修改**不会再生效**，请在 Web 控制台修改设置。
- 更新前备份整个 `data/` 目录，并保留 Docker 数据卷。
- Docker 更新使用 `docker compose pull && docker compose up -d`。
- 若更换 Bot Token，建议删除 `data/session.db` 后重启，避免旧会话不匹配。

## 常见问题

**没有发生导入**

确认配置文件路径正确、内容仍是合法 TOML，并检查启动日志。导入标记只会在导入成功后写入；已有同名存储或配置组不会被覆盖。

**数据库路径不同**

v1 不会把 v0 的 `[db] path` 当作新数据库路径。请用 `--db-path` / `SAVEANY_DB_PATH` 指向旧文件，或移动文件。

**本地存储路径失效**

v0 的本地存储路径相对于进程工作目录。Docker 中通常是 `/app`，请确认 `downloads/` 仍挂载到同一路径。

**登录密码未知**

查看首次启动日志中的 `Generated web admin password`。密码只保存哈希，无法从数据库还原；如果日志已丢失，请先备份数据库，再删除其中的 `auth` 设置并重启，让 v1 重新生成密码。

## 相关文档

- [安装与更新](../installation)
- [Web 控制台配置](../configuration/)
- [使用帮助](../../usage/)
