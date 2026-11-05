---
title: "命令行子命令"
weight: 21
---

# 命令行子命令

除直接运行 `./saveany-bot` 启动 Telegram Bot 外，二进制文件还提供两个辅助子命令用于上传本地文件：`upload`（一次性上传）和 `watch`（持续监听目录并上传）。

子命令读取相同的数据库配置与缓存，但不会启动 Bot 服务。当目标存储为 `telegram` 时，会在上传期间临时启动 Bot 客户端。

## `upload` — 上传单个文件

```
saveany-bot upload -f <文件> -s <存储名> [-d <目录>] [--no-progress]
```

参数:

| 参数 | 必填 | 说明 |
|---|---|---|
| `-f, --file` | 是 | 待上传的本地文件路径 |
| `-s, --storage` | 是 | 目标存储名 (必须在 Web 控制台的存储页面中存在) |
| `-d, --dir` | 否 | 存储中的目标目录, 默认使用存储的 `base_path` |
| `--no-progress` | 否 | 关闭终端进度条 |

示例:

```bash
# 上传文件到 "MyAlist" 的默认目录
./saveany-bot upload -f ./movie.mp4 -s MyAlist

# 上传到指定子目录
./saveany-bot upload -f ./movie.mp4 -s MyAlist -d movies/2026

# 通过 Telegram 存储上传并关闭进度条
./saveany-bot upload -f ./photo.jpg -s MyChannel --no-progress
```

## `watch` — 监听目录并自动上传

`watch` 子命令持续监听一个本地目录, 将新建或修改的文件上传到存储后端, 并保留相对监听根目录的子目录结构.

```
saveany-bot watch -p <路径> -s <存储名> [-d <目录>] [选项]
```

参数:

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-p, --path` | *(必填)* | 要监听的本地目录 |
| `-s, --storage` | *(必填)* | 目标存储名 |
| `-d, --dir` | 存储的 `base_path` | 存储中的目标目录 |
| `-r, --recursive` | `false` | 是否递归监听子目录 |
| `--overwrite` | `false` | 覆盖存储上已有的文件, 而非跳过 |
| `--initial-scan` | `false` | 启动时将目录中已存在的文件也上传 |
| `--debounce` | `2s` | 文件最后一次写入后, 等待多久再上传 |
| `--upload-workers` | `config.workers` | 并发上传数 |
| `--retry-delay` | `3s` | 上传重试之间的延迟 |

{{< hint info >}}
写入完成检测: 监听器会按文件做防抖处理, 仅当文件大小在一个 debounce 窗口内保持不变时才上传, 因此不会上传未写完的半成品文件.
<br />
若某文件在上传过程中又被修改, 它会在当前上传完成后再上传一次, 而不是被重复排队.
{{< /hint >}}

示例:

```bash
# 递归监听 ./inbox 并且把新文件上传到 "MyAlist"
./saveany-bot watch -p ./inbox -s MyAlist -r

# 自定义目标目录并覆盖已有文件
./saveany-bot watch -p ./inbox -s MyAlist -d backup --overwrite

# 启动时把 ./inbox 中已有的内容也一并上传
./saveany-bot watch -p ./inbox -s MyAlist --initial-scan
```

### 行为说明

- 相对子目录结构会被保留: 以 `--path ./inbox` 为例, 写入 `./inbox/sub/file.txt` 的文件会被上传到 `<目标目录>/sub/file.txt`.
- `watch` 会一直运行直到被中断 (如 `Ctrl-C` / `SIGINT`), 退出前会等待所有进行中的上传完成.
- 重试次数遵循 设置中的全局 `retry` 值, 各次重试之间间隔 `--retry-delay`.
- `telegram` 类型的存储会自动启动 Bot 客户端来执行上传.

{{< hint warning >}}
`watch` 子命令用于监听本地文件目录，与 Telegram 聊天的 `/watch` 命令无关。
{{< /hint >}}