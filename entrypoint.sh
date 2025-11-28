#!/bin/sh

if [ -n "$CONFIG_URL" ]; then
    echo "[INFO] Downloading config from $CONFIG_URL"
    if curl -sSLo /app/config.toml "$CONFIG_URL"; then
        echo "[INFO] Configuration downloaded successfully"
    else
        echo "[ERROR] Failed to download config from $CONFIG_URL"
        exit 1
    fi
fi

if [ ! -f /app/config.toml ]; then
    echo "[ERROR] Missing config.toml: Please provide the configuration file via mounting or CONFIG_URL"
    exit 1
fi
    
exec /app/saveany-bot