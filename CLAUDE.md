# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SaveAny-Bot is a Telegram bot written in Go that saves files from Telegram and external websites to various storage backends (Local, Alist, S3, WebDAV, Telegram). It supports bypassing "restrict saving content" media, batch downloads, auto-organization, and JS parser plugins for extracting content from websites.

## Commands

### Development
```bash
# Run the bot directly
go run ./cmd

# Run with specific config file
go run ./cmd --config /path/to/config.toml

# Generate i18n keys after modifying locale files
go generate ./...

# Run tests
go test ./...

# Run specific package tests
go test ./storage/s3
go test ./pkg/queue
```

### Docker
```bash
# Build and run locally
docker-compose -f docker-compose.local.yml up

# Production deployment
docker-compose up -d
```

## Architecture

### Application Initialization Flow (CRITICAL)

The application follows a strict initialization sequence in `cmd/run.go::initAll`:

1. **Config** (`config.Init`) - Load config.toml using viper
2. **Cache** (`common/cache`) - Initialize in-memory cache
3. **i18n** (`common/i18n`) - Load localization files from `common/i18n/locale/`
4. **Database** (`database.Init`) - Connect to SQLite, auto-migrate models
5. **Storage** (`storage.LoadStorages`) - Initialize all configured storage backends
6. **Parsers** (`parsers.LoadPlugins`) - Load JS parser plugins if enabled
7. **Userbot** (optional) - Login to Telegram userbot if configured
8. **Bot** (`client/bot.Init`) - Initialize Telegram bot client
9. **Core** (`core.Run`) - Start task queue workers

**When adding new initialization steps, add them to `initAll` in this order.**

### Core Components

#### Task Queue System (`core/`)
- Central abstraction: `core.Executable` interface with `Type()`, `Title()`, `TaskID()`, `Execute(ctx)`
- Task queue: `pkg/queue.TaskQueue[Executable]` managed by `core.Run`
- Worker pool size controlled by `config.Workers`
- Task implementations in `core/tasks/`:
  - `tfile/` - Telegram file downloads
  - `batchtfile/` - Batch downloads
  - `telegraph/` - Telegraph page saving
  - `parsed/` - Content from parser plugins
  - `directlinks/` - Direct URL downloads
- Tasks support lifecycle hooks: `TaskBeforeStart`, `TaskSuccess`, `TaskFail`, `TaskCancel` (configured via `config.Hook.Exec`)

#### Telegram Client (`client/`)
- **Bot**: `client/bot/bot.go` uses `gotgproto.NewClient`, handlers in `client/bot/handlers/`
- **Userbot**: `client/user/` for bypassing restricted content (optional)
- **Middleware**: `client/middleware/` handles flood wait, recovery, retry
- Handler registration: All handlers registered in `handlers.Register()`, command list in `handlers.CommandHandlers`

#### Storage Backends (`storage/`)
- Abstract interface: `Storage` with `Init()`, `Save()`, `Exists()`, `JoinStoragePath()`
- Implementations: `local/`, `alist/`, `s3/`, `minio/`, `webdav/`, `telegram/`
- Storage registry: `storage.Storages` map, constructed via `storageConstructors`
- Add new storage: Define enum in `pkg/enums/storage`, config in `config/storage/`, implementation in `storage/`, register in `storageConstructors`

#### Parser Plugins (`parsers/`)
- Native parsers: `parsers/native/twitter/`, `parsers/native/kemono/`
- JS plugin runtime: `parsers/js/` using `goja` VM and `playwright-go`
- Plugin interface: `registerParser({ metadata, canHandle, parse })` in JS
- Plugins return `parser.Item` with `Resources[]`, converted to `core/tasks/parsed` tasks
- Plugin docs: `plugins/README.md`

#### Configuration (`config/`)
- Main config: `config/viper.go::Config` loaded via `viper`
- Storage configs: `config/storage/factory.go::LoadStorageConfigs` with type-specific validation
- Environment override: Prefix `SAVEANY_`, dots become underscores (e.g., `SAVEANY_TELEGRAM_TOKEN`)
- **Important**: `config.C()` returns a copy, don't modify its fields; modify via `viper` in `config.Init`

#### Database (`database/`)
- SQLite with GORM
- Models: `User`, `Dir`, `Rule`, `WatchChat`
- User sync: `database.syncUsers` syncs `config.Users` to DB (creates/deletes users based on config)
- Don't create/delete users manually; use config file

#### Internationalization (`common/i18n/`)
- Locale files: `common/i18n/locale/*.yaml`
- Key generation: `go generate ./...` runs `cmd/geni18n/main.go` to generate `common/i18n/i18nk/keys.go`
- Usage: `i18n.T(i18nk.KeyName, map[string]any{"Param": value})`
- Add new strings: Edit YAML → run `go generate` → use new key in code

#### Logging
- Uses `github.com/charmbracelet/log`
- Logger injected into context in `cmd/run.go::Run` via `log.WithContext`
- Prefer `log.FromContext(ctx)` over global logger when context is available

### Data Flow

1. User sends message/file to Telegram bot
2. Handler in `client/bot/handlers/` processes update
3. Handler creates task (implements `core.Executable`) and calls `core.AddTask()`
4. Task enters `pkg/queue.TaskQueue`
5. Worker goroutine picks up task, calls `Execute(ctx)`
6. Task downloads content (from Telegram or parser) and saves to storage backend via `Storage.Save()`
7. Task completion triggers hooks, sends result message back to user

For parsed content (URLs):
1. Handler extracts URL, calls `parsers.ParseWithContext()`
2. Parser (native or JS plugin) returns `parser.Item` with `Resources[]`
3. `core/tasks/parsed.Task` created with resources
4. Task downloads each resource and saves to storage

## Project-Specific Guidelines

### Adding New Storage Backend

1. Define enum in `pkg/enums/storage/storage.go`
2. Create config struct in `config/storage/yourtype.go` implementing `StorageConfig` with `Validate()`
3. Implement storage in `storage/yourtype/yourtype.go` with `Storage` interface
4. Register in `config/storage/factory.go::storageFactories` and `storage/storage.go::storageConstructors`
5. Update `config.example.toml` with example configuration

### Adding New Task Type

1. Create package in `core/tasks/yourtype/`
2. Define struct implementing `core.Executable` interface
3. Implement `Type()`, `Title()`, `TaskID()`, `Execute(ctx)`
4. Add task type enum in `pkg/enums/tasktype/`
5. Create task from handler and call `core.AddTask(ctx, task)`

### Adding New Bot Command

1. Create handler file in `client/bot/handlers/yourcommand.go`
2. Implement handler function with signature matching gotgproto patterns
3. Register in `handlers.Register()` function
4. Add to `handlers.CommandHandlers` slice for command list
5. Add i18n strings for command description and responses

### Adding i18n Strings

1. Edit locale files in `common/i18n/locale/` (e.g., `zh-Hans.yaml`, `en.yaml`)
2. Run `go generate ./...` to generate `common/i18n/i18nk/keys.go`
3. Use in code: `i18n.T(i18nk.YourNewKey, map[string]any{"Variable": value})`

### Working with Middleware

New cross-cutting concerns (logging, metrics, rate limiting) should be implemented as middleware in `client/middleware/`:
1. Create middleware file with function signature matching gotgproto middleware
2. Add to `middleware.NewDefaultMiddlewares()` in `client/bot/bot.go`

### Parser Plugin Development

- JS plugins go in directories specified by `config.parser.plugin_dirs` (default: `./plugins/`)
- See `plugins/README.md` for plugin API documentation
- Plugins have access to `ghttp` for HTTP requests and `playwright` for browser automation
- Test plugins in `./testplugins/` directory
