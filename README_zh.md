<div align="center">

# <img src="docs/static/logo.svg" width="45" align="center"> Save Any Bot

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
- 在不同存储端之间转存文件
- 集成 yt-dlp, 从所支持的网站下载并转存媒体文件
- 集成 Aria2, 支持直链/磁力下载和转存
- 使用 js 编写解析器插件以转存任意网站的文件
- 存储端支持:
  - Alist
  - S3
  - WebDAV
  - 本地磁盘
  - Rclone
  - Telegram (重传回指定聊天)

## 快速开始

SaveAny-Bot 在 Web 控制台中配置。使用空的数据、下载和缓存目录启动:

```bash
docker run -d --name saveany-bot \
    -p 8080:8080 \
    -v ./data:/app/data \
    -v ./downloads:/app/downloads \
    -v ./cache:/app/cache \
    ghcr.io/krau/saveany-bot:latest
```

打开 `http://localhost:8080`。首次启动的管理员密码会打印在日志中:

```bash
docker logs saveany-bot | grep "admin password"
```

也可以通过 `SAVEANY_ADMIN_PASSWORD` 指定。登录后，指南页面会引导完成必要配置:

1. 连接 Telegram Bot。
2. 添加存储位置。
3. 授权你的 Telegram 账号。

完成后，向 Bot 发送文件或消息链接，即可在任务页查看转存进度。

使用预编译二进制时，下载 Release、解压并直接运行。程序会创建 `data/` 目录并输出管理员密码:

```bash
./saveany-bot
```

更多存储端和使用方法请 [**查看文档**](https://sabot.unv.app/)。

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