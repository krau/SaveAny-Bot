# SaveAny-Bot HTTP API

This document describes all currently registered HTTP API endpoints in SaveAny-Bot.

## Base URL

- Default: `http://127.0.0.1:28080`
- The API shares the Web console listener. Configure the listener and access token under **Settings → Web**.

## Authentication

When an access token is configured under **Settings → Web**, requests require:

```http
Authorization: Bearer <your-access-token>
```

Without a token, use the Web console session cookie or the tokenized URL printed at startup. Non-GET requests made with
a cookie also require the `X-CSRF-Token` header.

Missing or invalid authentication returns `401 Unauthorized`:

```json
{
  "error": "unauthorized",
  "message": "authentication required"
}
```

## Common Response Format

### Success

- `Content-Type: application/json`
- Status code depends on endpoint

### Error

```json
{
  "error": "error_code",
  "message": "human readable message"
}
```

## Task Status Values

- `queued`
- `running`
- `completed`
- `partial` — the task finished, but one or more files failed
- `failed`
- `cancelled`

Tasks that handle multiple files expose per-file outcomes. List responses include the total failed file count in
`failed_count` and up to five failed items per task in `failed_items`; the single-task endpoint returns every recorded
file in `items`.

## Endpoints

## 1) Health Check

- **Path**: `/health`
- **Method**: `GET`
- **Description**: Check API service health.

### Request

- Headers: none
- Body: none

### Response

- `200 OK`

```json
{
  "status": "ok"
}
```

### Example

```bash
curl -X GET "http://127.0.0.1:28080/health" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 2) Create Task

- **Path**: `/api/v1/tasks`
- **Method**: `POST`
- **Description**: Create and enqueue a task.

### Request

- Headers:
  - `Content-Type: application/json`
  - `Authorization: Bearer <token>` (only if an access token is configured)

### Request Body

```json
{
  "type": "directlinks",
  "storage": "local-main",
  "path": "downloads/movies",
  "webhook": "https://example.com/webhook",
  "params": {}
}
```

Field details:

- `type` (string, required):
  - `directlinks`
  - `ytdlp`
  - `aria2`
  - `parseditem`
  - `tgfiles`
  - `tphpics`
  - `transfer`
- `storage` (string, required): target storage name from configured storages
- `path` (string, optional): target path on storage
- `webhook` (string, optional): callback URL for task completion/failure
- `params` (object, required): type-specific payload

#### `params` by `type`

1. `directlinks`

```json
{
  "urls": ["https://example.com/file.zip"]
}
```

2. `ytdlp`

```json
{
  "urls": ["https://www.youtube.com/watch?v=dQw4w9WgXcQ"],
  "flags": ["--format", "bestvideo+bestaudio"]
}
```

3. `aria2`

```json
{
  "urls": ["https://example.com/large-file.iso"],
  "options": {
    "max-connection-per-server": "8"
  }
}
```

4. `parseditem`

```json
{
  "url": "https://example-parser-site.com/post/123"
}
```

5. `tgfiles`

```json
{
  "message_links": [
    "https://t.me/channel_name/100",
    "https://t.me/channel_name/101"
  ]
}
```

6. `tphpics`

```json
{
  "telegraph_url": "https://telegra.ph/example-page-01-01"
}
```

7. `transfer`

```json
{
  "source_storage": "s3-source",
  "source_path": "backup/2026-03-01",
  "target_storage": "local-main",
  "target_path": "restored"
}
```

### Response

- `201 Created`

```json
{
  "task_id": "d4f7u1crvimcs4n4j8d0",
  "type": "directlinks",
  "status": "queued",
  "created_at": "2026-03-10T10:00:00Z"
}
```

### Error Examples

- `400 Bad Request` invalid JSON:

```json
{
  "error": "invalid_request",
  "message": "failed to decode request body: ..."
}
```

- `400 Bad Request` unsupported type/storage/params:

```json
{
  "error": "task_creation_failed",
  "message": "unsupported task type: unknown"
}
```

### Example

```bash
curl -X POST "http://127.0.0.1:28080/api/v1/tasks" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "directlinks",
    "storage": "local-main",
    "path": "downloads",
    "params": {
      "urls": ["https://example.com/file.zip"]
    }
  }'
```

---

## 3) List Tasks

- **Path**: `/api/v1/tasks`
- **Method**: `GET`
- **Description**: List all tracked API tasks, newest first (by creation time).

### Request

- Headers:
  - `Authorization: Bearer <token>` (only if an access token is configured)
- Body: none

### Response

- `200 OK`

```json
{
  "tasks": [
    {
      "task_id": "d4f7u1crvimcs4n4j8d0",
      "type": "directlinks",
      "status": "running",
      "title": "Direct Links Download",
      "progress": {
        "total_bytes": 104857600,
        "downloaded_bytes": 52428800,
        "total_files": 3,
        "downloaded_files": 1,
        "percent": 50,
        "speed_mbps": 12.5
      },
      "storage": "local-main",
      "path": "downloads",
      "created_at": "2026-03-10T10:00:00Z",
      "updated_at": "2026-03-10T10:03:00Z"
    }
  ],
  "total": 1
}
```

### Example

```bash
curl -X GET "http://127.0.0.1:28080/api/v1/tasks" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 4) Get Task By ID

- **Path**: `/api/v1/tasks/{task_id}`
- **Method**: `GET`
- **Description**: Retrieve one task's metadata and progress.

### Request

- Headers:
  - `Authorization: Bearer <token>` (only if an access token is configured)
- Path parameter:
  - `task_id` (required)
- Body: none

### Response

- `200 OK`

```json
{
  "task_id": "d4f7u1crvimcs4n4j8d0",
  "type": "directlinks",
  "status": "running",
  "title": "Direct Links Download",
  "progress": {
    "total_bytes": 104857600,
    "downloaded_bytes": 52428800,
    "total_files": 3,
    "downloaded_files": 1,
    "percent": 50,
    "speed_mbps": 12.5
  },
  "storage": "local-main",
  "path": "downloads",
  "created_at": "2026-03-10T10:00:00Z",
  "updated_at": "2026-03-10T10:03:00Z"
}
```

### Error Examples

- `404 Not Found`

```json
{
  "error": "task_not_found",
  "message": "task not found: d4f7u1crvimcs4n4j8d0"
}
```

### Example

```bash
curl -X GET "http://127.0.0.1:28080/api/v1/tasks/d4f7u1crvimcs4n4j8d0" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 5) Retry Failed Files

- **Path**: `/api/v1/tasks/{task_id}/retry`
- **Method**: `POST`
- **Description**: Create a new task from the failed files of a terminal task. The source messages are fetched again,
  which also refreshes their file references.
- **Supported**: Telegram file tasks (`tgfiles`) whose files were saved with a recorded source message.
- **Request**: empty body. Bearer token (or session cookie with CSRF header) as usual.

### Response

- `202 Accepted`

```json
{
  "task_id": "d4f7u1crvimcs4n4j8e1",
  "status": "queued",
  "file_count": 2
}
```

### Error Examples

- `409 task_active` — the task is still queued or running
- `409 nothing_to_retry` — the task has no failed or unfinished files
- `409 storage_missing` — the target storage no longer exists
- `409 source_unavailable` — a failed file has no recorded source message
- `400 retry_unsupported` — the task type cannot be retried
- `503 client_unavailable` — no Telegram client is running

```bash
curl -X POST "http://127.0.0.1:28080/api/v1/tasks/d4f7u1crvimcs4n4j8d0/retry" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 6) Cancel Task

- **Path**: `/api/v1/tasks/{task_id}`
- **Method**: `DELETE`
- **Description**: Cancel a queued/running task.

### Request

- Headers:
  - `Authorization: Bearer <token>` (only if an access token is configured)
- Path parameter:
  - `task_id` (required)
- Body: none

### Response

- `200 OK`

```json
{
  "message": "task cancelled successfully"
}
```

### Error Examples

- `404 Not Found`

```json
{
  "error": "task_not_found",
  "message": "task not found: d4f7u1crvimcs4n4j8d0"
}
```

- `500 Internal Server Error`

```json
{
  "error": "cancel_failed",
  "message": "failed to cancel task: ..."
}
```

### Example

```bash
curl -X DELETE "http://127.0.0.1:28080/api/v1/tasks/d4f7u1crvimcs4n4j8d0" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 7) List Storages

- **Path**: `/api/v1/storages`
- **Method**: `GET`
- **Description**: List loaded storage backends.

### Request

- Headers:
  - `Authorization: Bearer <token>` (only if an access token is configured)
- Body: none

### Response

- `200 OK`

```json
{
  "storages": [
    {
      "name": "local-main",
      "type": "local"
    },
    {
      "name": "my-s3",
      "type": "s3"
    }
  ]
}
```

### Example

```bash
curl -X GET "http://127.0.0.1:28080/api/v1/storages" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 8) List Supported Task Types

- **Path**: `/api/v1/task-types`
- **Method**: `GET`
- **Description**: List supported task type values for task creation.

### Request

- Headers:
  - `Authorization: Bearer <token>` (only if an access token is configured)
- Body: none

### Response

- `200 OK`

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

### Example

```bash
curl -X GET "http://127.0.0.1:28080/api/v1/task-types" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 9) Get Media Metadata

- **Path**: `/api/v1/media-metadata`
- **Method**: `GET`
- **Description**: Inspect a supported media URL and return available metadata such as title, thumbnail, uploader, and audio/video duration.

Supported inputs:

- Telegram message links such as `https://t.me/username/123` and `https://t.me/c/123456789/123`
- Direct audio/video file URLs that `ffprobe` can inspect
- Other media page URLs supported by `yt-dlp`

### Request

- Headers:
  - `Authorization: Bearer <token>` (only if an access token is configured)
- Query parameters:
  - `url` (required): Telegram message link, direct media URL, or media page URL supported by yt-dlp

### Response

- `200 OK`

```json
{
  "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "title": "Rick Astley - Never Gonna Give You Up (Official Video)",
  "thumbnail": "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
  "uploader": "Rick Astley",
  "duration_seconds": 213.0
}
```

`title`, `thumbnail` and `uploader` are omitted when the source does not
expose them. `duration_seconds` is always present.

### Error Examples

- `400 Bad Request` missing URL:

```json
{
  "error": "invalid_request",
  "message": "url query parameter is required"
}
```

- `400 Bad Request` unsupported or unreadable URL:

```json
{
  "error": "metadata_extraction_failed",
  "message": "failed to inspect media: ..."
}
```

### Example

```bash
curl -X GET "http://127.0.0.1:28080/api/v1/media-metadata?url=https%3A%2F%2Fwww.youtube.com%2Fwatch%3Fv%3DdQw4w9WgXcQ" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## Webhook Callbacks

When `webhook` is set on a create-task request, a `POST` is sent to that URL
once the task reaches `completed`, `partial`, or `failed`. Cancelled tasks do not fire a
callback, and each task fires at most one. Delivery is asynchronous, so a slow
endpoint never blocks the task.

Headers:

```http
Content-Type: application/json
User-Agent: SaveAny-Bot/1.0
```

Body:

```json
{
  "task_id": "d4f7u1crvimcs4n4j8d0",
  "type": "directlinks",
  "status": "completed",
  "storage": "local-main",
  "path": "downloads",
  "completed_at": "2026-03-10T10:05:00Z",
  "error": ""
}
```

`completed_at` is present only for `completed` and `failed`. `error` is present
only when non-empty.

Up to 3 attempts are made. A non-2xx response or a transport error retries
after 100ms, then 400ms. Each request times out after 30 seconds.

---

## Task Retention

Task records live in memory only and are lost on restart. A task that has
reached a terminal state is also evicted 24 hours after its last update; the
sweep runs every 10 minutes.

## Notes

- Unknown endpoint returns:

```json
{
  "error": "not_found",
  "message": "endpoint not found: /your/path"
}
```

- Wrong method on an endpoint returns:

```json
{
  "error": "method_not_allowed",
  "message": "method not allowed: PUT"
}
```
