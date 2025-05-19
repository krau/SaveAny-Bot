#!/bin/sh

# 远程配置下载逻辑（仅在变量存在时触发）
if [ -n "$CONFIG_URL" ]; then
    echo "[INFO] Downloading config from $CONFIG_URL"
    if curl -sSLo /app/config.toml "$CONFIG_URL"; then
        echo "[INFO] Configuration downloaded successfully"
    else
        echo "[ERROR] Failed to download config from $CONFIG_URL"
        exit 1
    fi
fi

# 检查配置文件是否存在
if [ ! -f /app/config.toml ]; then
    echo "[ERROR] Missing config.toml: 请通过挂载或 CONFIG_URL 提供配置文件"
    exit 1
fi

# 启动主程序
exec /app/saveany-bot
