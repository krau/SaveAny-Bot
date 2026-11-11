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

## 特性

- 支持文档、视频、图片、贴纸及 [Telegraph](https://telegra.ph/) 图集
- 支持转存受限频道中禁止转发与保存的内容
- 批量下载
- 流式传输
- 多用户支持
- 基于规则自动分类整理
- 监听指定聊天并按过滤器自动转存
- 存储端之间互相传输文件
- 集成 yt-dlp，下载视频并转存
- 集成 Aria2，支持直链与磁力链接转存
- 支持编写 JavaScript 解析器插件扩展站点
- 存储端支持:
  - Alist
  - S3
  - WebDAV
  - 本地磁盘
  - Rclone
  - Telegram (重传回指定聊天)

## 快速开始

SaveAny-Bot 通过 Web 控制台配置。启动时挂载数据、下载和缓存目录：

```bash
docker run -d --name saveany-bot \
    -p 28080:28080 \
    -e SAVEANY_LISTEN=0.0.0.0:28080 \
    -v ./data:/app/data \
    -v ./downloads:/app/downloads \
    -v ./cache:/app/cache \
    ghcr.io/krau/saveany-bot:v1-beta
```

Web 界面默认监听 `127.0.0.1:28080`。启动日志会输出带访问 Token 的直达地址：

```bash
docker logs saveany-bot | grep -E "Web UI available at|Generated web admin password"
```

通过 Docker 端口映射访问时，日志中的 `127.0.0.1` 指宿主机；如果 Docker 运行在另一台机器上，请将 `127.0.0.1` 替换为那台机器的 IP。

`v1-beta` 是持续指向最新 v1 Beta 的移动 tag；需要可复现部署时，请改用具体的 beta 版本 tag。

直接运行二进制文件时，可通过命令行指定监听地址：

```bash
./saveany-bot --listen 0.0.0.0:28080
```

访问该地址即可登录。首次启动日志同时包含初始管理员密码，登录后可在侧栏修改。若未在“设置 → Web”中指定 Token，每次启动均会重新生成。初次登录可根据指南页面完成配置：

1. 连接 Telegram Bot。
2. 添加存储位置。
3. 授权 Telegram 账号。

完成后向 Bot 发送文件或消息链接，即可在任务页查看转存进度。

若使用预编译二进制文件，解压后直接运行，程序将自动创建 `data/` 目录并输出访问地址：

```bash
./saveany-bot
```

更多存储配置与使用方法请参阅[官方文档](https://sabot.unv.app/)。

## 赞助

如果这个项目对你有帮助, 你可以考虑通过以下方式赞助我:

- [爱发电](https://afdian.com/a/unvapp)
- TRON: `TDyuEkV6qChgFbMRtveLiLcGWrF5hEYLap`
- EVM: `0xAB77832Ed3E17A5967A8bA3618A7c9bD022f548c`

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