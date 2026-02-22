# HTTP API

SaveAny-Bot provides an HTTP API for programmatic access to download files from Telegram messages.

## Configuration

Add the following section to your `config.toml`:

```toml
[api]
enable = true
host = "0.0.0.0"
port = 8080
token = "your-secret-token"  # Optional: set to require authentication
```

## Authentication

If `token` is configured, include it in requests:

- As `Authorization` header: `Authorization: Bearer your-secret-token`
- As query parameter: `?token=your-secret-token`

## Endpoints

### POST /api/v1/download

Download files from a Telegram message.

**Request Body:**

```json
{
  "url": "https://t.me/c/123456789/123",
  "storage": "local",
  "path": "optional/subdirectory"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| url | string | Yes | Telegram message link (public channel/group or private chat) |
| storage | string | Yes | Storage name as configured in config.toml |
| path | string | No | Subdirectory path to save the file |

**Response:**

```json
{
  "success": true,
  "task_ids": ["abc123"]
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/download \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "url": "https://t.me/channel/123",
    "storage": "local"
  }'
```

### GET /api/v1/task/{task_id}

Get the status of a download task.

**Response:**

```json
{
  "success": true,
  "task": {
    "id": "abc123",
    "status": "completed",
    "progress": 100,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:01:00Z",
    "filename": "video.mp4"
  }
}
```

**Task Status:**

| Status | Description |
|--------|-------------|
| pending | Task is waiting in queue |
| running | Task is currently downloading |
| completed | Task finished successfully |
| failed | Task encountered an error |
| cancelled | Task was cancelled |

**Example:**

```bash
curl http://localhost:8080/api/v1/task/abc123 \
  -H "Authorization: Bearer your-token"
```

### GET /health

Health check endpoint.

**Response:**

```json
{
  "status": "ok"
}
```

## Error Responses

```json
{
  "success": false,
  "error": "error message"
}
```

## Notes

1. The API server must be enabled in configuration and the bot must be running
2. Storage names must match those configured in `config.toml`
3. Private chat message links require the bot to have access to the chat
4. Large files may take time to download; use the task status endpoint to monitor progress
