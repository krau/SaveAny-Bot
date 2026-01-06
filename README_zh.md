<div align="center">

# <img src="docs/static/logo.png" width="45" align="center"> Save Any Bot

> **把 Telegram 上的文件转存到多种存储端**

[![Release Date](https://img.shields.io/github/release-date/krau/saveany-bot?label=release)](https://github.com/krau/saveany-bot/releases)
[![tag](https://img.shields.io/github/v/tag/krau/saveany-bot.svg)](https://github.com/krau/saveany-bot/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/krau/saveany-bot/build-release.yml)](https://github.com/krau/saveany-bot/actions/workflows/build-release.yml)
[![Stars](https://img.shields.io/github/stars/krau/saveany-bot?style=flat)](https://github.com/krau/saveany-bot/stargazers)
[![Downloads](https://img.shields.io/github/downloads/krau/saveany-bot/total)](https://github.com/krau/saveany-bot/releases)
[![Issues](https://img.shields.io/github/issues/krau/saveany-bot)](https://github.com/krau/saveany-bot/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/krau/saveany-bot?label=pr)](https://github.com/krau/saveany-bot/pulls)
[![License](https://img.shields.io/github/license/krau/saveany-bot)](./LICENSE)

</div>

## 🎯 特性

- 支持文档/视频/图片/贴纸…甚至还有 [Telegraph](https://telegra.ph/)
- 破解禁止保存的文件
- 批量下载
- 流式传输
- 多用户使用
- 基于存储规则的自动整理
- 监听并自动转存指定聊天的消息, 支持过滤
- 使用 js 编写解析器插件以转存任意网站的文件
- 存储端支持:
  - Alist
  - S3
  - WebDAV
  - 本地磁盘
  - Telegram (重传回指定聊天)

## 快速开始

创建文件 `config.toml` 并填入以下内容:

```toml
[telegram]
token = "" # 你的 Bot Token, 在 @BotFather 获取
[telegram.proxy]
# 启用代理连接 telegram
enable = false
url = "socks5://127.0.0.1:7890"

[[storages]]
name = "本地磁盘"
type = "local"
enable = true
base_path = "./downloads"

[[users]]
id = 114514 # 你的 Telegram 账号 id
storages = []
blacklist = true
```

使用 Docker 运行 Save Any Bot:

```bash
docker run -d --name saveany-bot \
    -v ./config.toml:/app/config.toml \
    -v ./downloads:/app/downloads \
    ghcr.io/krau/saveany-bot:latest
```

请 [**查看文档**](https://sabot.unv.app/) 以获取更多配置选项和使用方法.

## 赞助

本项目受到 [YxVM](https://yxvm.com/) 与 [NodeSupport](https://github.com/NodeSeekDev/NodeSupport) 的支持.

如果这个项目对你有帮助, 你可以考虑通过以下方式赞助我:

- [爱发电](https://afdian.com/a/unvapp)

## 鸣谢

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- [tdl](https://github.com/iyear/tdl)
- All the dependencies, contributors, sponsors and users.

## 社区和关于作者

- [![通知群组](https://img.shields.io/badge/ProjectSaveAny-Group-blue)](https://t.me/ProjectSaveAny)
- [![讨论区](https://img.shields.io/badge/Github-Discussion-white)](https://github.com/krau/saveany-bot/discussions)
- [![个人频道](https://img.shields.io/badge/Krau-PersonalChannel-cyan)](https://t.me/acherkrau)