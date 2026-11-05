---
title: "测试"
weight: 10
---

# 测试

项目提供预置测试数据的本地环境，以及无需 Telegram 凭据的端到端测试。

## 预置环境

```bash
make testenv
```

命令会创建临时数据目录并输出：

- Web UI 地址
- 管理员密码（默认为 `saveany-test`）
- 预置的本地存储和白名单用户
- 用于创建直链下载任务的示例 URL

使用 `--keep` 保留生成的数据目录，或传入 `--data-dir` 复用已有目录。Telegram 保持停止状态，可直接测试 Web 控制台及非 Telegram 任务流程。

## 后端端到端测试

```bash
make e2e-api
```

测试会构建二进制，在随机端口启动预置环境并调用公开 API。覆盖认证与 CSRF、新手引导、存储与用户生命周期、设置更新、SPA 静态资源回退，以及下载直链文件到预置存储的任务。

测试位于 `e2e/`，使用 `e2e` build tag，普通 `go test ./...` 不会触发。

## Telegram 端到端测试

```bash
make e2e-telegram
```

该测试让 Bot handler、dispatcher、任务队列和存储实现运行在内存 `tg.Invoker` 之上。覆盖 `/start`、群聊准入、将 Telegram 文件保存到本地存储（含进度编辑）及上传到 Telegram 存储。全程无需真实 Telegram 账号或外部网络。

## 前端端到端测试

```bash
make e2e-web-setup   # 首次运行, 下载 Chromium
make e2e-web
```

启动预置环境并使用 Playwright 驱动 Web UI。覆盖登录、存储配置测试与创建，以及直链任务创建与完成状态校验。测试产物写入 `tmp/e2e-web-results/`。

Playwright 为 `web/ui` 的开发依赖，版本由 `pnpm-lock.yaml` 锁定。`make e2e-web-setup` 会自动安装匹配的 Chromium。

## 完整测试

```bash
make e2e
```

依次运行 API、Telegram 和前端浏览器测试（前端测试前会自动构建 Web UI）。假 Telegram 层覆盖了命令、下载、上传与群聊准入；若需测试线上 Telegram 真实行为、Userbot 登录及限流，仍需使用测试账号。
