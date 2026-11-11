<div align="center">

# <img src="docs/static/logo.svg" width="45" align="center"> Save Any Bot

**English** | [简体中文](./README_zh.md)

> **Save Any Telegram File to Anywhere 📂. Support restrict saving content and beyond telegram.**

[![Release Date](https://img.shields.io/github/release-date/krau/saveany-bot?label=release)](https://github.com/krau/saveany-bot/releases)
[![tag](https://img.shields.io/github/v/tag/krau/saveany-bot.svg)](https://github.com/krau/saveany-bot/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/krau/saveany-bot/build-release.yml)](https://github.com/krau/saveany-bot/actions/workflows/build-release.yml)
[![Stars](https://img.shields.io/github/stars/krau/saveany-bot?style=flat)](https://github.com/krau/saveany-bot/stargazers)
[![Downloads](https://img.shields.io/github/downloads/krau/saveany-bot/total)](https://github.com/krau/saveany-bot/releases)
[![Issues](https://img.shields.io/github/issues/krau/saveany-bot)](https://github.com/krau/saveany-bot/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/krau/saveany-bot?label=pr)](https://github.com/krau/saveany-bot/pulls)
[![License](https://img.shields.io/github/license/krau/saveany-bot)](./LICENSE)

</div>

## 🎯 Features

- Support documents / videos / photos / stickers… and even [Telegraph](https://telegra.ph/)
- Bypass "restrict saving content" media
- Batch download
- Streaming transfer
- Multi-user support
- Auto organize files based on storage rules
- Watch specified chats and auto-save messages, with filters
- Transfer files between different storage backends
- Integrate with yt-dlp to download and save media from 1000+ websites
- Aria2 integration to download files from URLs/magnets and save to storages
- Write JS parser plugins to save files from almost any website
- Storage backends:
  - Alist
  - S3
  - WebDAV
  - Local filesystem
  - Rclone (via command line)
  - Telegram (re-upload to specified chats)

## 📦 Quick Start

SaveAny-Bot is configured in the web console. Start it with empty data, downloads, and cache directories:

```bash
docker run -d --name saveany-bot \
    -p 28080:28080 \
    -e SAVEANY_LISTEN=0.0.0.0:28080 \
    -v ./data:/app/data \
    -v ./downloads:/app/downloads \
    -v ./cache:/app/cache \
    ghcr.io/krau/saveany-bot:v1-beta
```

The web UI listens on `127.0.0.1:28080` by default. The startup log prints a direct URL with an access token:

```bash
docker logs saveany-bot | grep -E "Web UI available at|Generated web admin password"
```

The logged `127.0.0.1` URL is the host-side address when the port is published. If Docker runs on another machine, replace `127.0.0.1` with that machine's IP.

`v1-beta` is the moving beta channel. Pin an exact image tag when you need a reproducible deployment.

For a binary deployment, pass the address directly:

```bash
./saveany-bot --listen 0.0.0.0:28080
```

Open the URL to sign in. On the first start, the same log also contains the generated admin password; use it to change the password from the sidebar after signing in. Unless an access token is set under **Settings → Web**, a new token is generated on every start. The Guide page covers the required setup:

1. Connect your Telegram bot.
2. Add a storage destination.
3. Allow your Telegram account.

Then send files or message links to the bot and watch them in the Tasks page.

To run the binary, download a release, extract it, and start it. It creates `data/` and prints the same tokenized URL:

```bash
./saveany-bot
```

Please [**read the docs**](https://sabot.unv.app/en/) for storage backends and usage.

## Sponsors

If this project is helpful to you, consider sponsoring me via:

- [Afdian](https://afdian.com/a/unvapp)
- TRON: `TDyuEkV6qChgFbMRtveLiLcGWrF5hEYLap`
- EVM: `0xAB77832Ed3E17A5967A8bA3618A7c9bD022f548c`

## Thanks To

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- [tdl](https://github.com/iyear/tdl)
- All the dependencies, contributors, sponsors and users.

## Contact

- [![Group](https://img.shields.io/badge/ProjectSaveAny-Group-blue)](https://t.me/ProjectSaveAny)
- [![Discussion](https://img.shields.io/badge/Github-Discussion-white)](https://github.com/krau/saveany-bot/discussions)
- [![PersonalChannel](https://img.shields.io/badge/Krau-PersonalChannel-cyan)](https://t.me/acherkrau)