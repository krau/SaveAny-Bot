---
title: "测试"
weight: 10
---

# 测试

项目提供一个预置数据的本地环境和两套无需 Telegram 凭据的端到端测试.

## 预置环境

```bash
make testenv
```

命令会创建一个临时数据目录并打印:

- Web UI 地址
- 管理员密码 (默认为 `saveany-test`)
- 预置的本地存储和白名单用户
- 一个示例 URL, 可直接用于创建直链下载任务

使用 `--keep` 保留生成的数据目录, 或传入 `--data-dir` 复用已有目录. Telegram
保持停止状态, 因此可以立即测试 Web 控制台和非 Telegram 任务流程.

## 后端端到端测试

```bash
make e2e-api
```

测试会构建二进制, 在随机端口启动预置环境, 用真实 HTTP 客户端调用公开 API.
覆盖认证与 CSRF, 新手引导, 存储与用户生命周期, 设置更新, SPA 静态资源回退, 以及
从本地 HTTP 服务下载文件到预置存储的直链任务.

测试位于 `e2e/`, 使用 `e2e` build tag, 因此 `go test ./...` 不会启动该环境.

## Telegram 端到端测试

```bash
make e2e-telegram
```

该套测试让真实 Bot handler, dispatcher, 任务队列和存储实现运行在内存
`tg.Invoker` 之上. 覆盖 `/start`, 群聊准入, 将 Telegram 文件保存到本地存储
(包括进度编辑), 以及上传到 Telegram 存储. 全程不需要 Telegram 账号或网络:
gotd 的 client, sender, downloader 和 uploader 都在假 RPC 层之上原样运行.

## 前端端到端测试

```bash
make e2e-web-setup   # 首次运行, 下载 Chromium
make e2e-web
```

测试会启动预置环境, 并用 Playwright 驱动构建后的 Web UI. 覆盖登录, 存储测试与
创建, 以及从前端创建直链任务并等待完成. 测试产物写入 `tmp/e2e-web-results/`.

Playwright 是 `web/ui` 的开发依赖, 版本由 `pnpm-lock.yaml` 锁定.
`make e2e-web-setup` 会安装匹配的 Chromium.

## 一起运行

```bash
make e2e
```

依次运行 API, Telegram 和浏览器测试 (浏览器测试前会构建 Web UI). 假 Telegram
层已覆盖命令, 下载, 上传和群聊准入; 真实 Telegram 服务端语义, userbot 登录和线上
限流仍需要测试账号.
