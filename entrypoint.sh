#!/bin/sh

# 自动创建数据库目录，防止 SQLite 初始化失败
mkdir -p /app/data

# Download config from URL if provided
if [ -n "$CONFIG_URL" ]; then
    echo "[INFO] 正在从下载配置 $CONFIG_URL"
    if curl -sSLo /app/config.toml "$CONFIG_URL"; then
        echo "[INFO] 配置下载成功"
    else
        echo "[ERROR] 无法从下载配置 $CONFIG_URL"
        exit 1
    fi
fi

# Check if config file exists
if [ ! -f /app/config.toml ]; then
    echo "[ERROR] Missing config.toml: 请通过挂载或 CONFIG_URL 提供配置文件"
    exit 1
fi

# 更新配置文件中的Redis环境变量（如果已设置）
# 允许Docker环境变量覆盖配置文件设置
if [ -n "$REDIS_ADDR" ]; then
    echo "[INFO] 从环境设置Redis地址: $REDIS_ADDR"
    # Use sed to update or add redis_addr in the [db] section
    if grep -q "redis_addr" /app/config.toml; then
        sed -i "s|redis_addr = .*|redis_addr = \"$REDIS_ADDR\"|" /app/config.toml
    else
        # Add redis_addr after the [db] section
        sed -i '/^\[db\]/a redis_addr = "'"$REDIS_ADDR"'"' /app/config.toml
    fi
fi

if [ -n "$REDIS_USER" ]; then
    echo "[INFO] 从环境设置Redis用户名"
    if grep -q "redis_user" /app/config.toml; then
        sed -i "s|redis_user = .*|redis_user = \"$REDIS_USER\"|" /app/config.toml
    else
        sed -i '/^\[db\]/a redis_user = "'"$REDIS_USER"'"' /app/config.toml
    fi
fi

if [ -n "$REDIS_PASSWORD" ]; then
    echo "[INFO] 从环境设置Redis密码"
    if grep -q "redis_password" /app/config.toml; then
        sed -i "s|redis_password = .*|redis_password = \"$REDIS_PASSWORD\"|" /app/config.toml
    else
        sed -i '/^\[db\]/a redis_password = "'"$REDIS_PASSWORD"'"' /app/config.toml
    fi
fi

if [ -n "$REDIS_DB" ]; then
    echo "[INFO] 从环境设置Redis数据库: $REDIS_DB"
    if grep -q "redis_db" /app/config.toml; then
        sed -i "s|redis_db = .*|redis_db = $REDIS_DB|" /app/config.toml
    else
        sed -i '/^\[db\]/a redis_db = '"$REDIS_DB" /app/config.toml
    fi
fi

# Display database configuration info
if [ -n "$REDIS_ADDR" ]; then
    echo "[INFO] 使用Redis数据库: $REDIS_ADDR"
else
    echo "[INFO] 使用SQLite数据库（未配置Redis）"
fi

# Start the application
exec /app/saveany-bot
