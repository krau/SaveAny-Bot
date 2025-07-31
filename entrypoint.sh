#!/bin/sh

# Download config from URL if provided
if [ -n "$CONFIG_URL" ]; then
    echo "[INFO] Downloading config from $CONFIG_URL"
    if curl -sSLo /app/config.toml "$CONFIG_URL"; then
        echo "[INFO] Configuration downloaded successfully"
    else
        echo "[ERROR] Failed to download config from $CONFIG_URL"
        exit 1
    fi
fi

# Check if config file exists
if [ ! -f /app/config.toml ]; then
    echo "[ERROR] Missing config.toml: 请通过挂载或 CONFIG_URL 提供配置文件"
    exit 1
fi

# Update Redis environment variables in config file if they are set
# This allows Docker environment variables to override config file settings
if [ -n "$REDIS_ADDR" ]; then
    echo "[INFO] Setting Redis address from environment: $REDIS_ADDR"
    # Use sed to update or add redis_addr in the [db] section
    if grep -q "redis_addr" /app/config.toml; then
        sed -i "s|redis_addr = .*|redis_addr = \"$REDIS_ADDR\"|" /app/config.toml
    else
        # Add redis_addr after the [db] section
        sed -i '/^\[db\]/a redis_addr = "'"$REDIS_ADDR"'"' /app/config.toml
    fi
fi

if [ -n "$REDIS_PASSWORD" ]; then
    echo "[INFO] Setting Redis password from environment"
    if grep -q "redis_password" /app/config.toml; then
        sed -i "s|redis_password = .*|redis_password = \"$REDIS_PASSWORD\"|" /app/config.toml
    else
        sed -i '/^\[db\]/a redis_password = "'"$REDIS_PASSWORD"'"' /app/config.toml
    fi
fi

if [ -n "$REDIS_DB" ]; then
    echo "[INFO] Setting Redis database from environment: $REDIS_DB"
    if grep -q "redis_db" /app/config.toml; then
        sed -i "s|redis_db = .*|redis_db = $REDIS_DB|" /app/config.toml
    else
        sed -i '/^\[db\]/a redis_db = '"$REDIS_DB" /app/config.toml
    fi
fi

# Display database configuration info
if [ -n "$REDIS_ADDR" ]; then
    echo "[INFO] Using Redis database at: $REDIS_ADDR"
else
    echo "[INFO] Using SQLite database (Redis not configured)"
fi

# Start the application
exec /app/saveany-bot