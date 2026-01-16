# SaveAny-Bot Agent Guidelines

This document provides essential information for AI coding agents working on the SaveAny-Bot project.

## Project Overview

SaveAny-Bot is a Telegram bot written in Go that saves files/messages from Telegram and various websites to multiple storage backends (local, S3, MinIO, WebDAV, AList, Telegram). It features a plugin system for parsing web content and extensible storage backends.

**Tech Stack**: Go 1.24.2, gotd/td (Telegram MTProto), Cobra (CLI), Viper (config), GORM (ORM), SQLite, Goja (JS runtime), Playwright (browser automation)

## Build & Test Commands

### Build
```bash
# Standard build
go build -o saveany-bot .

# Run directly
go run ./cmd

# Docker build (multi-stage, Alpine-based)
docker build -t saveany-bot .
docker compose up -d
```

### Test
```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./pkg/queue
go test ./storage/telegram

# Run tests with verbose output
go test -v ./...

# Run a single test
go test -run TestQueueBasic ./pkg/queue

# Run with coverage
go test -cover ./...
```

### Lint & Format
```bash
# Format code (standard Go formatting)
go fmt ./...

# Vet code for common issues
go vet ./...

# Generate code (i18n keys)
go generate ./...
```

### Other Commands
```bash
# Update dependencies
go mod tidy

# View documentation
cd docs && hugo server -D
```

## Code Style Guidelines

### Imports
- Standard library first, then third-party, then project-internal
- Group imports with blank lines between groups
- Use explicit import aliases for clarity when needed (e.g., `storconfig`, `storenum`)

```go
import (
    "context"
    "fmt"
    
    "github.com/charmbracelet/log"
    
    "github.com/krau/SaveAny-Bot/config"
    "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)
```

### Formatting
- Line length: reasonable (no hard limit, but be sensible)
- Organize code with blank lines between logical sections
- Follow standard Go conventions for braces, spacing, etc.

### Types & Interfaces
- Use clear, descriptive type names (PascalCase for exported, camelCase for unexported)
- Define interfaces where abstraction is needed (e.g., `Executable`, `StorageConfig`)
- Embed context in method signatures, not structs: `func (s *Service) Do(ctx context.Context) error`
- Prefer composition over inheritance

```go
// Interfaces define behavior
type Executable interface {
    Type() tasktype.TaskType
    Title() string
    TaskID() string
    Execute(ctx context.Context) error
}

// Structs compose behavior
type Local struct {
    config config.LocalStorageConfig
    logger *log.Logger
}
```

### Naming Conventions
- **Packages**: lowercase, single word when possible (avoid underscores)
- **Files**: lowercase with underscores for multiword (e.g., `auth_terminal.go`, `progress_reader.go`)
- **Variables**: camelCase for unexported, PascalCase for exported
- **Constants**: PascalCase for exported, camelCase for unexported (not ALL_CAPS)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Test files**: `*_test.go` pattern

### Error Handling
- Always handle errors explicitly; never ignore them
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use `errors.Is()` and `errors.As()` for error checking
- Log errors with appropriate level (Error, Warn, Info)
- Return errors from functions rather than panicking (except for truly unrecoverable situations)

```go
// Good error handling
if err := db.Save(user).Error; err != nil {
    return fmt.Errorf("failed to save user %d: %w", user.ChatID, err)
}

// Check specific errors
if errors.Is(err, context.Canceled) {
    logger.Info("Operation was canceled")
    return nil
}
```

### Logging
- Use `github.com/charmbracelet/log` package
- Get logger from context: `log.FromContext(ctx)`
- Create prefixed loggers for components: `logger.WithPrefix("component")`
- Use appropriate levels: Debug, Info, Warn, Error
- Include context in log messages (e.g., task IDs, file names)

```go
logger := log.FromContext(ctx)
logger.Infof("Processing task: %s", task.ID)
logger.Errorf("Failed to save file %s: %v", filename, err)
```

### Concurrency
- Use channels for communication between goroutines
- Protect shared state with `sync.Mutex` or `sync.RWMutex`
- Use `sync.WaitGroup` for coordinating goroutine completion
- Always pass `context.Context` for cancellation support
- Use `context.WithCancel/WithTimeout` for managing goroutine lifetimes

```go
// Example from queue implementation
func (tq *TaskQueue[T]) Add(task *Task[T]) error {
    tq.mu.Lock()
    defer tq.mu.Unlock()
    // ... critical section
    tq.cond.Signal()
    return nil
}
```

### Comments
- Document exported types, functions, and packages with doc comments
- Start doc comments with the name being documented
- Use `//` for single-line comments
- Explain *why*, not *what* (code should be self-explanatory for "what")
- Add `[NOTE]`, `[WARN]`, `[IMPORTANT]` tags for important clarifications

```go
// GetUserByChatID retrieves a user by their Telegram chat ID.
// Returns an error if the user is not found.
func GetUserByChatID(ctx context.Context, chatID int64) (*User, error) {
```

## Architecture & Conventions

### Application Structure
- **Entry point**: `main.go` → `cmd.Execute(ctx)`
- **CLI root**: `cmd/root.go` (Cobra), implementation in `cmd/run.go`
- **Startup sequence**: Config → Cache → i18n → Database → Storage → Parsers → Userbot → Bot → Queue
- Follow this order when adding new initialization steps in `cmd/run.go::initAll`

### Configuration (Viper)
- Config defined in `config/viper.go::Config`
- Read from `config.toml` (see `config.example.toml`)
- Environment variables: `SAVEANY_*` prefix (e.g., `SAVEANY_TELEGRAM_TOKEN`)
- Access via `config.C()` (returns a copy, don't modify the return value)
- Storage configs validated via `config/storage/factory.go::LoadStorageConfigs`

### Telegram Client
- **Bot client**: `client/bot/bot.go::Init` (uses gotgproto)
- **Handlers**: Centralized in `client/bot/handlers/` directory
- **Registration**: All handlers registered in `handlers.Register`
- **Commands**: Add to `CommandHandlers` slice for automatic `/help` and bot command list updates
- **Middleware**: Common middleware in `client/middleware/` (floodwait, retry, etc.)

### Tasks & Queue
- **Task interface**: `core/core.go::Executable` (Type, Title, TaskID, Execute methods)
- **Queue**: `pkg/queue.TaskQueue[Executable]` (generic, thread-safe)
- **Workers**: Count from `config.C().Workers`
- **Task types**: Implementations in `core/tasks/**` (tfile, parsed, telegraph, directlinks, batchtfile)
- **Lifecycle hooks**: `TaskBeforeStart`, `TaskSuccess`, `TaskFail`, `TaskCancel` (defined in config)
- **Adding tasks**: Use `core.AddTask(ctx, task)`

### Database (GORM + SQLite)
- **Init**: `database.Init` using `config.C().DB.Path`
- **Models**: User, Dir, Rule, WatchChat (in `database/*.go`)
- **Migrations**: Automatic via `db.AutoMigrate`
- **User sync**: `database.syncUsers` syncs DB with `config.C().Users` (don't manually create/delete users)
- **Context**: Always use `db.WithContext(ctx)` for operations

### Storage Backends
- **Interface**: Defined in `config/storage/types.go` and `storage/`
- **Implementations**: local, alist, s3/minio, webdav, telegram (each in subdirectory)
- **Adding new storage**:
  1. Add enum to `pkg/enums/storage`
  2. Create config struct in `config/storage/` with `Validate()` method
  3. Implement storage in `storage/<name>/`
  4. Register in `storageFactories` mapping
  5. Update `config.example.toml` with example

### Parser Plugins (JavaScript)
- **Runtime**: Goja (JS runtime) + Playwright (browser automation)
- **Plugin API**: `registerParser({ metadata, canHandle, parse })` in JS
- **Integration**: Defined in `parsers/` directory
- **Documentation**: See `plugins/README.md`
- Plugin `parse` returns `Item`/`Resource` which becomes download/transfer task

### Internationalization (i18n)
- **Usage**: `i18n.T(i18nk.SomeKey, map[string]any{"Name": value})`
- **Locale files**: `common/i18n/locale/*.yaml`
- **Key generation**: Run `go generate ./...` to generate `common/i18n/i18nk/keys.go`
- **Adding new strings**: Add to YAML → run `go generate` → use in code
- All user-facing strings should be internationalized

### Context Usage
- Always pass `context.Context` as first parameter
- Use `log.FromContext(ctx)` to get contextual logger
- Respect context cancellation in long-running operations
- Store request-scoped data in context (e.g., `ctxkey.ContentLength`)

## Special Rules from .github/copilot-instructions.md

1. **Never modify `config.C()` return values** - it returns a copy. Modify config in `config.Init` or via Viper.
2. **Handlers must update `CommandHandlers` slice** - ensures `/help` and bot commands stay in sync.
3. **Task execution must preserve hooks** - don't remove `TaskBeforeStart`, `TaskSuccess`, `TaskFail`, `TaskCancel` hook calls.
4. **User sync is automatic** - don't manually create/delete users in DB; use config-based sync.
5. **Prefer context logger** - use `log.FromContext(ctx)` over global logger when context is available.
6. **Storage factory pattern** - new storage types must register in `storageFactories` mapping.
7. **Plugin API compatibility** - changes to `Item`/`Resource` structures require updating `plugins/README.md`.

## Common Patterns

### Adding a New Command
1. Create handler function in `client/bot/handlers/<name>.go`
2. Add to `CommandHandlers` slice in `register.go`
3. Add i18n key to `common/i18n/locale/*.yaml`
4. Run `go generate ./...`
5. Test with Telegram bot

### Adding a New Task Type
1. Create struct implementing `core.Executable` in `core/tasks/<type>/`
2. Implement `Type()`, `Title()`, `TaskID()`, `Execute(ctx)` methods
3. Add task type enum to `pkg/enums/tasktype`
4. Use `core.AddTask(ctx, task)` to enqueue

### Adding a New Storage Backend
1. Define config struct in `config/storage/<name>.go` with `Validate()` method
2. Implement storage interface in `storage/<name>/<name>.go`
3. Add storage type enum to `pkg/enums/storage`
4. Register factory in `config/storage/factory.go::storageFactories`
5. Update `config.example.toml` with configuration example

## File References

When referencing code locations, use `path/to/file.go:line` format (e.g., `core/core.go:23` for the worker function).

## Testing Guidelines

- Write tests for new functionality (place in `*_test.go` files)
- Test files should be in same package as code being tested
- Use table-driven tests for multiple test cases
- Mock external dependencies (databases, network calls)
- Aim for meaningful tests, not just coverage numbers

## Notes

- Binary size matters: use `CGO_ENABLED=0` for static binaries
- FFmpeg is included in Docker images for media processing
- Build process supports cross-compilation (amd64/arm64, Linux/macOS/Windows)
- Documentation site uses Hugo; edit files in `docs/` directory
- Session data stored in SQLite; delete `data/session.db` if changing bot token
