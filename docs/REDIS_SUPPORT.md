# Redis Database Support

This document describes the Redis database support added to SaveAny-Bot.

## Overview

SaveAny-Bot now supports both SQLite (default) and Redis for data storage. The system automatically selects the database backend based on configuration.

## Configuration

### SQLite (Default)
```toml
[db]
path = "data/saveany.db"
session = "data/session.db"
```

### Redis
```toml
[db]
path = "data/saveany.db"           # Still used for Telegram session storage
session = "data/session.db"        # Still used for Telegram session storage
redis_addr = "localhost:6379"      # Redis server address
redis_password = ""                # Redis password (optional)
redis_db = 0                       # Redis database number (default: 0)
```

## Docker Environment Variables

When running in Docker, you can use environment variables to configure Redis:

```bash
docker run -e REDIS_ADDR=redis:6379 -e REDIS_PASSWORD=mypassword -e REDIS_DB=0 saveany-bot
```

Environment variables:
- `REDIS_ADDR`: Redis server address
- `REDIS_PASSWORD`: Redis password (optional)
- `REDIS_DB`: Redis database number (optional, default: 0)

## How It Works

1. **Automatic Selection**: If `redis_addr` is configured, Redis is used; otherwise, SQLite is used
2. **Backward Compatibility**: Existing SQLite configurations continue to work without changes
3. **Transparent Operations**: All database operations work identically with both backends
4. **Data Structure**: Redis stores data as JSON documents with the same structure as SQLite

## Redis Data Structure

- **Users**: `users:{chatId}` → JSON of user data
- **Directories**: `dirs:{userId}:{dirId}` → JSON of directory data
- **Rules**: `rules:{userId}:{ruleId}` → JSON of rule data
- **Indexes**: `user_dirs:{userId}` and `user_rules:{userId}` → Sets of IDs for relationships

## Migration

To migrate from SQLite to Redis:

1. Configure Redis in your `config.toml`
2. Restart the application
3. The system will automatically use Redis for new data
4. Existing SQLite data won't be automatically migrated (manual migration would be needed if required)

## Benefits of Redis

- **Performance**: Faster read/write operations for high-traffic scenarios
- **Scalability**: Better support for horizontal scaling
- **Memory**: In-memory storage for faster access
- **Persistence**: Redis can persist data to disk when configured
- **Clustering**: Support for Redis clusters for high availability

## Requirements

- Redis server 5.0 or later
- Network connectivity to Redis server
- Appropriate Redis configuration for persistence (if data durability is required)