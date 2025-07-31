# Redis 数据库支持文档

本文档介绍 SaveAny-Bot 新增的 Redis 数据库支持功能。

## 概述

SaveAny-Bot 现已支持 SQLite（默认）和 Redis 双数据库存储引擎，系统会根据配置自动选择数据库后端。

## 配置说明

### SQLite（默认配置）
```toml
[db]
path = "data/saveany.db"
session = "data/session.db"
```

### Redis 配置
```toml
[db]
path = "data/saveany.db"           # 仍用于 Telegram 会话存储
session = "data/session.db"        # 仍用于 Telegram 会话存储
redis_addr = "localhost:6379"      # Redis 服务器地址
redis_user = ""                    # Redis ACL 用户名（可选，适用于 Redis 6+ ACL 认证）
redis_password = ""                # Redis 密码（可选）
redis_db = 0                       # Redis 数据库编号（默认：0）
```

## Docker 环境变量

在 Docker 环境中运行时，可通过环境变量配置 Redis：

```bash
docker run -e REDIS_ADDR=redis:6379 -e REDIS_USER=user -e REDIS_PASSWORD=mypassword -e REDIS_DB=0 saveany-bot
```

支持的环境变量：
- `REDIS_ADDR`: Redis 服务器地址
- `REDIS_PASSWORD`: Redis 密码（可选）
- `REDIS_DB`: Redis 数据库编号（可选，默认：0）

## 工作机制

1. **自动选择**：配置了 `redis_addr` 则使用 Redis，否则使用 SQLite
2. **向下兼容**：现有 SQLite 配置无需修改即可继续使用
3. **透明操作**：所有数据库操作在两种后端上表现一致
4. **数据结构**：Redis 以 JSON 文档形式存储数据，结构保持与 SQLite 相同

## Redis 数据结构

- **用户数据**：`users:{chatId}` → 用户数据 JSON
- **目录数据**：`dirs:{userId}:{dirId}` → 目录数据 JSON
- **规则数据**：`rules:{userId}:{ruleId}` → 规则数据 JSON
- **索引关系**：`user_dirs:{userId}` 和 `user_rules:{userId}` → 存储关联 ID 的集合

## 数据迁移

从 SQLite 迁移到 Redis 的步骤：

1. 在 `config.toml` 中配置 Redis
2. 重启应用
3. 系统将自动对新数据使用 Redis
4. 现有 SQLite 数据不会自动迁移（如需迁移需手动操作）

## Redis 优势

- **性能**：高并发场景下更快的读写速度
- **扩展性**：更好地支持水平扩展
- **内存存储**：内存级访问速度
- **持久化**：支持配置数据落盘
- **集群**：支持 Redis 集群实现高可用

## 系统要求

- Redis 服务器 5.0 及以上版本
- 可连接到 Redis 服务器的网络环境
- 如需数据持久化，需正确配置 Redis 持久化选项
