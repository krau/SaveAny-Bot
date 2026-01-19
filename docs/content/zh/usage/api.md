---
title: "HTTP API"
weight: 4
---

# HTTP API

SaveAny-Bot 提供 RESTful HTTP API，支持通过编程方式从 Telegram 下载文件。

## 配置

在 `config.toml` 中启用 API：

```toml
[api]
# 启用 HTTP API 服务
enable = true
# API 服务监听端口
port = 8080
# API 访问令牌 (留空则不验证)
token = "your-secret-token-here"
# 任务完成回调 Webhook URL (留空则不回调)
webhook_url = "https://your-server.com/webhook"
```

## 认证

如果配置了 `token`，所有 API 请求（除了 `/health`）都必须包含 `Authorization` 头：

```
Authorization: Bearer your-secret-token-here
```

## 端点

### 健康检查

检查 API 服务器是否正在运行。

**请求：**
```
GET /health
```

**响应：**
```json
{
  "status": "ok"
}
```

### 创建下载任务

从 Telegram 消息链接创建新的文件下载任务。

**请求：**
```
POST /api/v1/tasks
Content-Type: application/json
Authorization: Bearer your-secret-token-here

{
  "telegram_url": "https://t.me/channel/123",
  "user_id": 123456789,
  "storage_name": "local1",
  "dir_path": "/downloads"
}
```

**请求参数：**
- `telegram_url` (必填): Telegram 消息链接 (例如: `https://t.me/channel/123`)
- `user_id` (必填): Telegram 用户 ID (必须在 `config.toml` 中配置)
- `storage_name` (可选): 要使用的存储名称。如果未指定，使用用户的第一个可用存储
- `dir_path` (可选): 存储中的目录路径。默认为 `/`

**响应 (201 Created)：**
```json
{
  "task_id": "c9h8t1234abcd",
  "message": "task created successfully"
}
```

**错误响应 (4xx/5xx)：**
```json
{
  "error": "错误描述"
}
```

### 获取任务状态

获取特定任务的状态。

**请求：**
```
GET /api/v1/tasks/{task_id}
Authorization: Bearer your-secret-token-here
```

**响应 (200 OK)：**
```json
{
  "task_id": "c9h8t1234abcd",
  "status": "completed",
  "title": "[tgfiles](file.pdf->local1:/downloads/file.pdf)",
  "created_at": "2024-01-19T04:30:00Z",
  "error": ""
}
```

**状态值：**
- `queued`: 任务正在队列中等待
- `running`: 任务正在下载
- `completed`: 任务成功完成
- `failed`: 任务失败（查看 `error` 字段）
- `canceled`: 任务已取消

### 列出所有任务

列出所有排队和正在运行的任务。

**请求：**
```
GET /api/v1/tasks
Authorization: Bearer your-secret-token-here
```

**响应 (200 OK)：**
```json
{
  "queued": [
    {
      "id": "c9h8t1234abcd",
      "title": "[tgfiles](file1.pdf->local1:/downloads/file1.pdf)"
    }
  ],
  "running": [
    {
      "id": "d2k9u5678efgh",
      "title": "[tgfiles](file2.pdf->local1:/downloads/file2.pdf)"
    }
  ]
}
```

### 取消任务

取消正在运行或排队的任务。

**请求：**
```
DELETE /api/v1/tasks/{task_id}
Authorization: Bearer your-secret-token-here
```

**响应 (200 OK)：**
```json
{
  "message": "task canceled"
}
```

## Webhook 回调

如果配置了 `webhook_url`，API 会在任务完成或失败时向 webhook URL 发送 POST 请求。

**Webhook 请求：**
```
POST {webhook_url}
Content-Type: application/json
Authorization: Bearer your-secret-token-here

{
  "task_id": "c9h8t1234abcd",
  "status": "completed",
  "title": "[tgfiles](file.pdf->local1:/downloads/file.pdf)",
  "created_at": "2024-01-19T04:30:00Z",
  "error": ""
}
```

## 使用示例

### 使用 cURL

**创建下载任务：**
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-secret-token-here" \
  -d '{
    "telegram_url": "https://t.me/channel/123",
    "user_id": 123456789,
    "storage_name": "local1",
    "dir_path": "/downloads"
  }'
```

**获取任务状态：**
```bash
curl http://localhost:8080/api/v1/tasks/c9h8t1234abcd \
  -H "Authorization: Bearer your-secret-token-here"
```

**列出所有任务：**
```bash
curl http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer your-secret-token-here"
```

**取消任务：**
```bash
curl -X DELETE http://localhost:8080/api/v1/tasks/c9h8t1234abcd \
  -H "Authorization: Bearer your-secret-token-here"
```

### 使用 Python

```python
import requests

API_URL = "http://localhost:8080"
TOKEN = "your-secret-token-here"
HEADERS = {
    "Authorization": f"Bearer {TOKEN}",
    "Content-Type": "application/json"
}

# 创建下载任务
response = requests.post(
    f"{API_URL}/api/v1/tasks",
    headers=HEADERS,
    json={
        "telegram_url": "https://t.me/channel/123",
        "user_id": 123456789,
        "storage_name": "local1",
        "dir_path": "/downloads"
    }
)
task_id = response.json()["task_id"]

# 获取任务状态
response = requests.get(
    f"{API_URL}/api/v1/tasks/{task_id}",
    headers=HEADERS
)
status = response.json()
print(f"任务状态: {status['status']}")
```

## 安全建议

1. **生产环境始终使用强令牌**
2. **生产环境使用 HTTPS**，通过反向代理（如 Nginx、Caddy）放置 API
3. **保护日志安全**，因为它们可能包含敏感信息
4. **验证用户权限** - 确保请求中的 `user_id` 对应于配置中的授权用户
