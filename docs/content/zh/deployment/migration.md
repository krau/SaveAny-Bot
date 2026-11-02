---
title: "从 v0 升级"
weight: 15
---

# 从 v0 升级

v1 使用本地 SQLite 数据库保存配置。首次启动时，如果检测到 v0 的 `config.toml`，会把它导入数据库一次；之后所有配置都在 Web 控制台管理。

v1 不迁移 Telegram 会话。v0 的 `session.db` 和 `usersession.db` 由旧版 gotgproto 生成，v1 不再读取。迁移只导入旧版配置；v1 使用新的数据库。Bot 会使用当前 Token 重新登录，启用 Userbot 时需要在 Web 控制台重新登录。升级前请完整备份现有文件。

## 1. 停止并备份

先停止 v0，再备份配置、数据和下载目录：

```bash
docker compose down
cp -a config.toml data downloads /path/to/backup/
```

如果使用二进制部署，直接停止进程后备份相同目录即可。

不要同时运行 v0 和 v1，也不要让两个实例共用同一个 `data/` 目录。

## 2. 保留数据目录

v1 不复用 v0 数据库。请保留旧文件作为备份，由 v1 创建新的 `data/saveany.db`：

| v0 文件 | v1 默认路径 | 说明 |
|---|---|---|
| `data/saveany.db` | 不复用 | 保留作为备份；v1 创建新数据库 |

不要用 `--db-path` 或 `SAVEANY_DB_PATH` 指向 v0 数据库；v0 schema 不会迁移。请在首次启动 v1 前把旧的 `data/saveany.db` 移出数据目录或改名，v1 必须在默认路径创建新数据库。请保留旧文件作为备份。

v0 的 `data/session.db` 和 `data/usersession.db` 会被忽略，确认已备份后可删除。v1 使用新的会话文件：

| 文件 | 说明 |
|---|---|
| `data/bot_session.db` | Bot 会话 |
| `data/userbot_session.db` | Userbot 会话 |

Bot 使用当前 Token 重新登录。启用 Userbot 时，请稍后在 Telegram 页面重新登录。

## 3. 首次启动时导入配置

首次启动必须让 v1 能读到旧版 `config.toml`。Docker 只在这时挂载配置文件：

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

二进制部署：

```bash
./saveany-bot --config /path/to/config.toml
```

省略 `--config` 时，v1 会依次查找工作目录下的 `./config.toml` 和 `/etc/saveany/config.toml`。

日志中出现配置导入完成后，即可移除 `config.toml` 挂载并重启。v1 在数据库中写入导入标记，不会重复导入。`config.toml` 中的 `db.session` 和 `telegram.userbot.session` 会被忽略，v1 始终使用新的会话文件路径。

## 4. 登录并核对

启动日志会打印带访问 Token 的 Web UI 直达地址，直接打开即可登录。如果首次启动没有设置 `SAVEANY_ADMIN_PASSWORD`，备用管理员密码也会打印在日志中：

```bash
docker compose logs saveany-bot | grep -E "Web UI available at|Generated web admin password"
```

登录后打开 **指南**，确认：

- Bot 已使用当前 Token 重新连接。
- 若启用了 Userbot，已重新登录并显示账号。
- 存储已启用且没有初始化错误。
- 用户列表包含原有账号。v0 配置中不存在的旧用户会被禁用，需要时手动启用。
- 从 `config.toml` 导入的存储、用户和设置仍然存在。
- 目录、规则和监听需要在 v1 中重新创建；v0 数据库行不会迁移。

然后向 Bot 发送一个文件或 Telegram 消息链接，确认任务能正常完成。

## 5. 迁移之后

- `config.toml` 的后续修改**不会再生效**，请在 Web 控制台修改设置。
- MinIO 存储会在启动时转换为 `s3`，请在 Web 控制台核对 Endpoint、密钥和 Bucket。
- 更新前备份整个 `data/` 目录，并保留 Docker 数据卷。
- Docker 更新使用 `docker compose pull && docker compose up -d`。
- 若更换 Bot Token，建议删除 `data/bot_session.db` 后重启，避免旧会话不匹配。

## 常见问题

**没有发生导入**

确认配置文件路径正确、内容仍是合法 TOML，并检查启动日志。导入标记只会在导入成功后写入；已有同名存储或配置组不会被覆盖。

**旧会话没有迁移**

v1 不读取旧版 gotgproto 的 `session.db` 或 `usersession.db`。Bot 会使用当前 Token 自动重新登录；启用 Userbot 时，在 Telegram 页面重新登录即可。旧文件可以保留在备份中，确认不再需要后删除。

**旧数据库不会迁移**

v1 不复用 v0 数据库或其 `[db] path`。请保留旧文件作为备份，让 v1 创建新数据库。

**本地存储路径失效**

v0 的本地存储路径相对于进程工作目录。Docker 中通常是 `/app`，请确认 `downloads/` 仍挂载到同一路径。

**登录密码未知**

查看首次启动日志中的 `Generated web admin password`。密码只保存哈希，无法从数据库还原；如果日志已丢失，请先备份数据库，再删除其中的 `auth` 设置并重启，让 v1 重新生成密码。

## 相关文档

- [安装与更新](../installation)
- [Web 控制台配置](../configuration/)
- [使用帮助](../../usage/)
