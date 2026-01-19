---
title: "HTTP API"
weight: 4
---

# HTTP API

SaveAny-Bot provides a RESTful HTTP API for programmatic file downloads from Telegram.

## Configuration

Enable the API in your `config.toml`:

```toml
[api]
# Enable HTTP API service
enable = true
# API server listen port
port = 8080
# API access token (leave empty to disable authentication)
token = "your-secret-token-here"
# Task completion callback webhook URL (leave empty to disable)
webhook_url = "https://your-server.com/webhook"
# Trusted IP addresses (leave empty to allow all), supports single IP or CIDR notation
trusted_ips = ["127.0.0.1", "192.168.1.0/24"]
```

## Authentication

If `token` is configured, all API requests (except `/health`) must include an `Authorization` header:

```
Authorization: Bearer your-secret-token-here
```

If `trusted_ips` is configured, requests will only be accepted from specified IP addresses.

## Endpoints

### Health Check

Check if the API server is running.

**Request:**
```
GET /health
```

**Response:**
```json
{
  "status": "ok"
}
```

### Create Download Task

Create a new file download task from a Telegram message link.

**Request:**
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

**Request Parameters:**
- `telegram_url` (required): Telegram message link (e.g., `https://t.me/channel/123`)
- `user_id` (required): Telegram user ID (must be configured in `config.toml`)
- `storage_name` (optional): Storage name to use. If not specified, uses the first available storage for the user
- `dir_path` (optional): Directory path in storage. Default is `/`

**Response (201 Created):**
```json
{
  "task_id": "c9h8t1234abcd",
  "message": "task created successfully"
}
```

**Error Response (4xx/5xx):**
```json
{
  "error": "error description"
}
```

### Get Task Status

Get the status of a specific task.

**Request:**
```
GET /api/v1/tasks/{task_id}
Authorization: Bearer your-secret-token-here
```

**Response (200 OK):**
```json
{
  "task_id": "c9h8t1234abcd",
  "status": "completed",
  "title": "[tgfiles](file.pdf->local1:/downloads/file.pdf)",
  "created_at": "2024-01-19T04:30:00Z",
  "error": ""
}
```

**Status Values:**
- `queued`: Task is waiting in queue
- `running`: Task is currently downloading
- `completed`: Task completed successfully
- `failed`: Task failed with error (see `error` field)
- `canceled`: Task was canceled

### List All Tasks

List all queued and running tasks.

**Request:**
```
GET /api/v1/tasks
Authorization: Bearer your-secret-token-here
```

**Response (200 OK):**
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

### Cancel Task

Cancel a running or queued task.

**Request:**
```
DELETE /api/v1/tasks/{task_id}
Authorization: Bearer your-secret-token-here
```

**Response (200 OK):**
```json
{
  "message": "task canceled"
}
```

## Webhook Callback

If `webhook_url` is configured, the API will send a POST request to the webhook URL when a task completes or fails.

**Webhook Request:**
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

## Example Usage

### Using cURL

**Create a download task:**
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

**Get task status:**
```bash
curl http://localhost:8080/api/v1/tasks/c9h8t1234abcd \
  -H "Authorization: Bearer your-secret-token-here"
```

**List all tasks:**
```bash
curl http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer your-secret-token-here"
```

**Cancel a task:**
```bash
curl -X DELETE http://localhost:8080/api/v1/tasks/c9h8t1234abcd \
  -H "Authorization: Bearer your-secret-token-here"
```

### Using Python

```python
import requests

API_URL = "http://localhost:8080"
TOKEN = "your-secret-token-here"
HEADERS = {
    "Authorization": f"Bearer {TOKEN}",
    "Content-Type": "application/json"
}

# Create a download task
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

# Get task status
response = requests.get(
    f"{API_URL}/api/v1/tasks/{task_id}",
    headers=HEADERS
)
status = response.json()
print(f"Task status: {status['status']}")
```

## Security Recommendations

1. **Always use a strong token** for production environments
2. **Enable IP whitelist** (`trusted_ips`) to restrict access
3. **Use HTTPS** in production by placing the API behind a reverse proxy (e.g., Nginx, Caddy)
4. **Keep logs secure** as they may contain sensitive information
5. **Validate user permissions** - ensure `user_id` in requests corresponds to authorized users in your config
