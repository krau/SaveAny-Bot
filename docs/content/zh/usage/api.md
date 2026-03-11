---
title: "HTTP API"
weight: 20
---

# HTTP API

SaveAny-Bot 提供了一套 HTTP API，允许你通过程序化方式创建下载/转存任务、查询任务状态、取消任务等，无需通过 Telegram 操作。

## 启用 API

在 `config.toml` 中添加或修改以下配置：

```toml
[api]
enable = true
host   = "0.0.0.0"   # 监听地址，默认 0.0.0.0
port   = 8080         # 监听端口，默认 8080
token  = "your-token" # 鉴权 Token，强烈建议设置
```

也可通过环境变量覆盖（前缀 `SAVEANY_`）：

| 环境变量 | 对应配置项 |
|---|---|
| `SAVEANY_API_ENABLE` | `api.enable` |
| `SAVEANY_API_HOST` | `api.host` |
| `SAVEANY_API_PORT` | `api.port` |
| `SAVEANY_API_TOKEN` | `api.token` |

{{< hint warning >}}
若 `token` 为空，API 服务将**不进行任何鉴权**即可访问，存在安全风险。
{{< /hint >}}

## 鉴权

当配置了 `token` 时，所有 API 请求均需在 HTTP 请求头中携带 Bearer Token：

```
Authorization: Bearer <your-token>
```

鉴权失败时返回 `401`：

```json
{ "error": "unauthorized", "message": "invalid token" }
```

## 错误响应格式

所有错误均使用统一的 JSON 格式：

```json
{
  "error":   "error_code",
  "message": "错误说明"
}
```

常见错误码：

| 错误码 | HTTP 状态 | 含义 |
|---|---|---|
| `unauthorized` | 401 | 鉴权失败 |
| `method_not_allowed` | 405 | HTTP 方法不正确 |
| `invalid_request` | 400 | 请求体/参数非法 |
| `task_creation_failed` | 400 | 任务创建失败 |
| `task_not_found` | 404 | 任务 ID 不存在 |
| `cancel_failed` | 500 | 取消任务失败 |
| `internal_error` | 500 | 服务器内部错误 |

---

## 接口列表

### GET /health — 健康检查

无需鉴权。

**响应 `200 OK`：**

```json
{ "status": "ok" }
```

---

### GET /api/v1/storages — 列出存储

返回当前所有已加载的存储后端。

**响应 `200 OK`：**

```json
{
  "storages": [
    { "name": "local",   "type": "local" },
    { "name": "MyMinio", "type": "s3" }
  ]
}
```

---

### GET /api/v1/task-types — 列出支持的任务类型

**响应 `200 OK`：**

```json
{
  "types": [
    "directlinks",
    "ytdlp",
    "aria2",
    "parseditem",
    "tgfiles",
    "tphpics",
    "transfer"
  ]
}
```

---

### POST /api/v1/tasks — 创建任务

**请求头：**

```
Content-Type: application/json
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "type":    "<任务类型>",
  "storage": "<存储名>",
  "path":    "<子目录>",
  "webhook": "<回调URL>",
  "params":  { }
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `type` | string | 是 | 任务类型，见下文 |
| `storage` | string | 是 | 目标存储名，须与配置中的存储名一致 |
| `path` | string | 否 | 存储内的子目录路径 |
| `webhook` | string | 否 | 任务完成/失败时的回调地址 |
| `params` | object | 是 | 各任务类型的专属参数，见下文 |

**响应 `201 Created`：**

```json
{
  "task_id":    "abc123xyz",
  "type":       "directlinks",
  "status":     "queued",
  "created_at": "2026-03-11T10:00:00Z"
}
```

#### 任务类型与 params

##### directlinks — 直接下载链接

下载一个或多个 HTTP/HTTPS 直链文件。

```json
{
  "type":    "directlinks",
  "storage": "local",
  "path":    "downloads",
  "params": {
    "urls": [
      "https://example.com/file.zip",
      "https://example.com/other.zip"
    ]
  }
}
```

| params 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `urls` | []string | 是 | 下载地址列表，至少 1 条 |

##### ytdlp — yt-dlp 视频下载

{{< hint warning >}}
需要在系统中安装 yt-dlp。
{{< /hint >}}

通过 yt-dlp 下载视频/音频，支持 YouTube、Bilibili 等 1000+ 网站。

```json
{
  "type":    "ytdlp",
  "storage": "local",
  "path":    "videos",
  "params": {
    "urls":  ["https://www.youtube.com/watch?v=xxx"],
    "flags": ["--extract-audio", "--audio-format", "mp3"]
  }
}
```

| params 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `urls` | []string | 是 | 媒体链接列表，至少 1 条 |
| `flags` | []string | 否 | 额外的 yt-dlp 命令行参数 |

##### aria2 — Aria2 下载

{{< hint warning >}}
需要在配置文件中启用并配置 Aria2 RPC。
{{< /hint >}}

通过 Aria2 下载管理器下载文件，支持 HTTP/HTTPS、FTP、BitTorrent（磁力链接、种子）等协议。

```json
{
  "type":    "aria2",
  "storage": "local",
  "path":    "downloads",
  "params": {
    "urls":    ["magnet:?xt=urn:btih:..."],
    "options": { "split": "4" }
  }
}
```

| params 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `urls` | []string | 是 | 下载地址列表，至少 1 条 |
| `options` | map[string]string | 否 | Aria2 下载选项 |

##### parseditem — 解析器下载

将 URL 交由已注册的 JS 插件或内置解析器处理后下载。

```json
{
  "type":    "parseditem",
  "storage": "local",
  "path":    "parsed",
  "params": {
    "url": "https://some-site.com/page"
  }
}
```

| params 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `url` | string | 是 | 待解析的页面 URL |

若没有任何解析器能处理该 URL，则返回 `400 task_creation_failed`。

##### tgfiles — Telegram 消息文件下载

通过 Telegram 消息链接下载文件。支持以下链接格式：

- `https://t.me/username/123` — 公开频道/群组
- `https://t.me/c/123456789/123` — 私有频道（数字 ID）
- `https://t.me/c/123456789/111/456` — 话题消息
- `https://t.me/username/111/456` — 用户名频道下的话题消息

若消息属于媒体组（相册），默认下载整组文件。在链接末尾追加 `?single` 可强制只下载单条消息的文件。

```json
{
  "type":    "tgfiles",
  "storage": "local",
  "path":    "telegram",
  "params": {
    "message_links": [
      "https://t.me/username/123",
      "https://t.me/c/1234567890/456"
    ]
  }
}
```

| params 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `message_links` | []string | 是 | Telegram 消息链接列表，至少 1 条 |

##### tphpics — Telegraph 文章图片下载

下载 Telegra.ph 文章中的所有图片。

支持的链接前缀：`https://telegra.ph/`、`http://telegra.ph/`、`https://telegraph.co/`、`http://telegraph.co/`

```json
{
  "type":    "tphpics",
  "storage": "local",
  "path":    "telegraph",
  "params": {
    "telegraph_url": "https://telegra.ph/Some-Article-01-01"
  }
}
```

| params 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `telegraph_url` | string | 是 | Telegra.ph 文章 URL |

##### transfer — 存储间文件传输

在两个存储后端之间直接传输文件，无需经过 Telegram。源存储须支持列举（list）和读取（read）操作。

{{< hint info >}}
`transfer` 任务中，顶层的 `storage` 字段仍然必须填写（用于通过参数校验），但实际使用的存储由 `params` 中的 `source_storage` 和 `target_storage` 决定。
{{< /hint >}}

```json
{
  "type":    "transfer",
  "storage": "local",
  "params": {
    "source_storage": "MyS3",
    "source_path":    "backups/",
    "target_storage": "LocalDisk",
    "target_path":    "restored/"
  }
}
```

| params 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `source_storage` | string | 是 | 源存储名 |
| `source_path` | string | 是 | 源存储中的路径，须包含至少一个文件 |
| `target_storage` | string | 是 | 目标存储名 |
| `target_path` | string | 是 | 目标存储中的路径 |

---

### GET /api/v1/tasks — 列出所有任务

返回所有 API 创建的任务（仅在内存中保留，重启后清空）。

**响应 `200 OK`：**

```json
{
  "tasks": [
    {
      "task_id":    "abc123xyz",
      "type":       "directlinks",
      "status":     "running",
      "title":      "file.zip",
      "storage":    "local",
      "path":       "downloads",
      "error":      "",
      "created_at": "2026-03-11T10:00:00Z",
      "updated_at": "2026-03-11T10:00:05Z",
      "progress": {
        "total_bytes":      10485760,
        "downloaded_bytes": 5242880,
        "percent":          50.0
      }
    }
  ],
  "total": 1
}
```

`progress` 字段仅在 `total_bytes > 0` 时出现。`error` 字段仅在有错误时出现。

---

### GET /api/v1/tasks/{task_id} — 查询任务

**路径参数：** `task_id` — 创建任务时返回的 ID。

**响应 `200 OK`：** 同上列表中的单个任务对象。

**错误响应：**
- `400 invalid_request` — 路径中未提供 task_id
- `404 task_not_found` — 任务不存在

---

### DELETE /api/v1/tasks/{task_id} — 取消任务

**路径参数：** `task_id`

**响应 `200 OK`：**

```json
{ "message": "task cancelled successfully" }
```

**错误响应：**
- `400 invalid_request` — 路径中未提供 task_id
- `404 task_not_found` — 任务不存在
- `500 cancel_failed` — 取消操作失败

---

## 任务状态

| 状态值 | 含义 |
|---|---|
| `queued` | 已入队，等待执行 |
| `running` | 正在执行 |
| `completed` | 已成功完成 |
| `failed` | 执行失败 |
| `cancelled` | 已通过 DELETE 接口取消 |

---

## Webhook 回调

创建任务时可设置 `webhook` 字段。当任务进入终态（`completed`、`failed`、`cancelled`）时，Bot 会向该地址发送一个 `POST` 请求。

**回调请求头：**

```
Content-Type: application/json
User-Agent: SaveAny-Bot/1.0
```

**回调请求体：**

```json
{
  "task_id":      "abc123xyz",
  "type":         "directlinks",
  "status":       "completed",
  "storage":      "local",
  "path":         "downloads",
  "completed_at": "2026-03-11T10:01:00Z",
  "error":        ""
}
```

`completed_at` 仅在状态为 `completed` 或 `failed` 时出现。`error` 仅在有错误时出现。

**重试机制：** 最多重试 3 次，重试间隔依次为 1 秒、2 秒、3 秒。每次请求超时为 30 秒。
