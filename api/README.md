# API Module

This module provides a RESTful HTTP API for programmatic file downloads from Telegram.

## Features

- **RESTful API endpoints** for creating, querying, and canceling download tasks
- **Bearer token authentication** for API access control
- **IP whitelist** support for additional security
- **Webhook callbacks** for task completion notifications
- **Task status tracking** (queued, running, completed, failed, canceled)
- **Graceful shutdown** with proper cleanup

## Usage

See the full documentation at:
- English: `/docs/content/en/usage/api.md`
- Chinese: `/docs/content/zh/usage/api.md`

## Architecture

- `server.go` - HTTP server initialization and route registration
- `handlers.go` - API endpoint handlers and business logic
- `middleware.go` - Authentication and logging middleware

The API integrates with the existing task queue system and uses the bot's Telegram client to fetch messages.
