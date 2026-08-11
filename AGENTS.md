# Repository Guidelines

## Project Overview

SaveAny-Bot is a Telegram bot written in Go that saves files and messages from Telegram and websites to multiple storage backends (Local, S3, MinIO, WebDAV, AList, Rclone, Telegram). It supports single-file saves, batch/album saves, streaming, multi-user access, storage rules, cross-storage transfers, yt-dlp and Aria2 downloads, and a Goja-based JavaScript parser plugin system with optional Playwright browser automation.

**Tech stack**: Go 1.25, gotgproto + gotd/td v0.149.0 (MTProto), Cobra (CLI), Viper (config), GORM + SQLite, Goja (JS runtime), Playwright, charmbracelet/log. License: AGPL-3.0.

**Note on gotd versions**: `gotd/td` must stay on `v0.149.0` — v0.150+ breaks `gotgproto` (v1.0.0-beta22) compilation (`AsInputDocumentFileLocation` signature, `gotd/log` Logger interface). Do not bump beyond v0.149.

## Architecture & Data Flow

```
Telegram update → client/bot/handlers → core.AddTask(Executable) → pkg/queue (serial workers)
    → core/tasks/* (download via common/tdler) → storage.Storage → progress feedback
```

- **Startup sequence** (`cmd/run.go::initAll`, keep this order): Config → Cache → i18n → Database → Storage → Parser plugins → Userbot → API → Bot. `bot.Init` returns the exit channel; a `SAVEANTBOT-RESTART` error restarts the process (external supervisor).
- **Task pipeline**: handlers build a task (via `core.AddTask`), the queue executes `Executable{Type, Title, TaskID, Execute(ctx)}` with `config.C().Workers` workers. Lifecycle hooks (`TaskBeforeStart/Success/Fail/Cancel`) run around `Execute`. Cancellation = canceling the task's context; tasks must check `ctx.Err()`.
- **Dual progress channels**: (1) `pkg/taskevent` context bus (consumed by `api/` for HTTP/Webhook consumers — `taskevent.WithSink` must be injected for API tasks), (2) Telegram message edits via `ProgressTracker` + `tgutil.ExtFromContext(ctx)` (bot tasks). Upload progress currently reaches the Telegram channel only.
- **Capability interfaces + type assertion fallback** is the core extensibility pattern: `StorageBatchSaver`, `StorageProgressSaver` (upload progress), `StorageListable`, `StorageReadable` are optional; consumers assert and fall back (e.g. wrap reader with `ioutil.NewProgressReader`). New backends only need to implement the interface and register.
- **Config layering**: CLI flag > env `SAVEANY_*` (dots → underscores, e.g. `SAVEANY_TELEGRAM_TOKEN`) > TOML file (path or http(s) URL). `config.C()` returns a **copy** — never mutate it.
- **Storage registration (3 places)**: `pkg/enums/storage` ENUM comment (go-enum), `config/storage/factory.go::storageFactories` (config struct with `Validate()`), `storage/storage.go::storageConstructors` (implementation).

## Key Directories

| Path | Purpose |
|---|---|
| `cmd/` | CLI: `run` (main bot), `upload`, `watch` (standalone subcommands that do NOT run initAll), `geni18n` (i18n key generator) |
| `core/` | `Executable` interface, queue worker loop, hooks; `core/tasks/{tfile,batchtfile,directlinks,parsed,telegraph,transfer,ytdlp,aria2dl}` |
| `client/bot/` | gotgproto client, `handlers/` (all commands + message/callback handlers), `middleware/`, `client/user/` (userbot) |
| `storage/` | 8 backends + `storage.go` (interfaces/registry) + `load.go` (per-user storage resolution) |
| `parsers/` | `parsers.go` (registry), `js/` (Goja plugins, ghttp/playwright injection, build-tagged), `parsers/` (native: twitter, kemono) |
| `config/` | Viper setup, defaults, `storage/` per-type config structs |
| `database/` | GORM models (User/Dir/Rule/WatchChat), AutoMigrate, `syncUsers` |
| `pkg/` | `queue`, `taskevent`, `tcbdata` (callback data), `rule`, `enums/{tasktype,storage,ctxkey,fnamest}`, `storagetypes`, `tfile`, `parser` |
| `common/` | `tdler` (unified downloader), `utils/{tgutil,dlutil,ioutil,fsutil,strutil,tphutil,netutil}`, `i18n` (embedded locales), `cache` (ristretto) |
| `api/` | HTTP API + webhook (task factory with sink injection) |
| `docs/` | Hugo site (hugo-book theme, zh+en mirrored), separate go.mod |
| `plugins/` | JS parser examples + `README.md` (plugin author contract) |

## Development Commands

```bash
# Build (standard; CGO_ENABLED=0 for static)
CGO_ENABLED=0 go build -trimpath -o saveany-bot .
go run ./cmd

# Test — known failures: storage/telegram TestCreateSplitZip/TestExtractThumbFrame/TestGetVideoMetadata
# (need gitignored fixtures tests/testfile.dat, tests/testvideo; ffmpeg/ffprobe)
go test ./...
go test -race ./core/tasks/... ./storage/... ./pkg/queue/... ./common/...
go test -run TestQueueBasic ./pkg/queue

# Codegen — run after editing locale YAML or enum comments
go generate ./...            # geni18n (i18nk keys) + go-enum (pkg/enums/*)
# go-enum is NOT in go.mod; install externally. geni18n runs via go run.

# Verify
go vet ./...
go fmt ./...
```

**Build variants** (Dockerfile.default/micro/pico): `-tags=no_jsparser,no_playwright,no_minio,no_bubbletea,sqlite_glebarez` — each has a `*_stub.go`/`*_glebarez.go` pairing; keep stubs in sync.

Docker: `docker build -t saveany-bot .`, `docker compose up -d` (host network, mounts `./data ./config.toml ./downloads ./cache`). CI (`.github/workflows/`) runs **no tests/lint** — only tag-triggered release/docker builds and docs deployment; run `go test ./...` manually before pushing.

## Code Conventions & Common Patterns

- **Imports**: stdlib → third-party → project-internal, blank-line separated. Aliases for clarity (`storconfig`, `storenum`).
- **Naming**: PascalCase exported, camelCase unexported, files `snake_case.go`; **not** ALL_CAPS constants.
- **Errors**: always wrap with `fmt.Errorf("context: %w", err)`; check with `errors.Is/As`; never ignore.
- **Logging**: `log.FromContext(ctx)` with prefixes (`logger.WithPrefix("component")`); never global logger when ctx is available.
- **Context values** (read from the passed ctx, never globals): `log.FromContext`, `tgutil.ExtFromContext` (Telegram ext — **required for message edits; if nil, edits are silently dropped**), `storage.FromContext`, `storagetypes.WithSourceCaption`, `ctxkey.ContentLength` / `ctxkey.OverwriteExisting`.
- **Progress rendering** (#228 convention): i18n templates declare styles with Telegram HTML (`<b>/<code>/<blockquote>/<i>`); dynamic data MUST go through `i18n.T(key, tgutil.EscapeHTMLTemplateData(data))` before `tgutil.RenderHTML`. Never interpolate user data raw, never render-then-substring-search.
- **Progress tracking**: each task package defines its own small `ProgressTracker` interface; optional `UploadProgressTracker` is probed via type assertion (skip if absent). Serialize state + message edits with a mutex; throttle edits (≥1s); aggregate per-item progress monotonically.
- **i18n**: only edit `common/i18n/locale/{zh-Hans,en}.yaml` → `go generate ./...` → use `i18nk.<Key>` constants. No raw strings in user-facing messages. zh-Hans and en must stay in sync.
- **Registration points** (never forget): new bot command → `client/bot/handlers/register.go::CommandHandlers` (auto-publishes /help menu); new task type → `pkg/enums/tasktype` + `core/tasks/<name>/` + `api/factory.go::CreateTask`; new storage → 3 places above + `docs/content/{en,zh}/deployment/configuration/storages.md`; new enum value → ENUM comment + `go generate`.
- **Concurrency**: `errgroup.WithContext` + `SetLimit(config.C().Workers)`, `atomic.Int64` counters, `sync.Once` for single-shot events, mutex around render state. No lock-in-callback (callbacks fire after unlock).
- **Cancellation**: queue tasks carry a `WithCancel`-derived ctx; check `ctx.Err()` in loops; classify with `errors.Is(err, context.Canceled)`.
- **JS plugins**: `registerParser({metadata, canHandle, parse})`, `version >= 1.0.0`; per-plugin goja VM is single-goroutine (reqCh buffer 10). Changing `pkg/parser.Item/Resource` JSON fields requires updating `plugins/README.md` and example plugins.
- **Message edits**: `ext.EditMessage(chatID, &tg.MessagesEditMessageRequest{...})`; cancel buttons via `tgutil.BuildCancelButton(taskID)`; callback payloads via `pkg/tcbdata` + `common/cache`.

## Important Files

- `main.go` — `//go:generate` for i18n keys
- `cmd/run.go` — startup sequence `Run/initAll/cleanCache` (cache cleanup on exit, `NoCleanCache` opt-out)
- `core/core.go` — worker loop, hooks, AddTask/CancelTask
- `pkg/queue/queue.go` — generic serial queue (cond/list; duplicate TaskID rejected)
- `storage/storage.go` — interfaces + registry + compile-time capability assertions
- `config/viper.go`, `config.example.toml` — config schema (authoritative field docs)
- `database/db.go` — GORM init, `GetDialect` (build-tag selectable SQLite driver)
- `client/bot/handlers/register.go` — handler dispatch order and CommandHandlers
- `common/tdler/dler.go` — unified download entry
- `core/tasks/batchtfile/item_progress.go` — per-item phase state machine (Downloading/Transferring/Uploading/Retrying/Confirming, FailureStage)
- `parsers/js/plugin.go` — Goja plugin runtime
- `.github/workflows/` — release/docker/docs (no test gate)

## Runtime/Tooling Preferences

- **Go 1.25+**: `t.Context()`, `sync.WaitGroup.Go`, `for range n` are available.
- **Runtime binaries**: ffmpeg/ffprobe (media processing/video split), yt-dlp (ytdlp tasks), aria2 optional; Playwright browsers install on demand to `./playwright` (`playwright.Install(chromium, ...)` at first `pw.get()`); Docker images: default has ffmpeg+yt-dlp, micro only curl, pico is scratch static.
- **No Makefile, no golangci.yml, no test/lint CI** — verification is manual (`go vet`, `go test`).
- **go-enum** required externally for enum generation; **geni18n** is in-repo.
- **Docs**: Hugo site in `docs/` (separate go.mod, hugo-book); edit `docs/content/{zh,en}/` — keep both languages mirrored. `docs/public/` is gitignored build output.
- **gitignored fixtures**: `storage/telegram/tests/` (missing — 3 tests fail locally), `data/`, `config.toml`, `playwright/`, `testplugins/`.

## Testing & QA

- Pure stdlib `testing` (no testify); table-driven (`[]struct{name...}` + `t.Run`) with `t.Fatalf` got/want assertions. Mock via hand-written interface impls or package-variable replacement (`runMediaTool` in `video_split_test.go`, restored with `t.Cleanup`); in-process services for HTTP (`httptest`), S3 (`gofakes3+s3mem`), WebDAV (`x/net/webdav`).
- **Locale-dependent tests**: pin with `i18n.Init("zh-Hans")` + `t.Cleanup(...)`.
- **Progress/HTML tests**: assert rendered text with `strings.Contains` AND entity counts (`tg.MessageEntityBold/Code/Blockquote/Italic`) — verify style injection stays escaped (`<b>A&B</b>` input must render as literal text).
- **Known failures**: `storage/telegram` `TestCreateSplitZip`, `TestExtractThumbFrame`, `TestGetVideoMetadata` need gitignored fixtures + real ffmpeg — skip with `-skip 'Test(CreateSplitZip|ExtractThumbFrame|GetVideoMetadata)$'`; `api/handlers_test.go` has one `t.Skip` (needs initialized core).
- **Coverage expectations**: pure logic gets table tests (parsers, URL/path utils, progress throttling, grouping); regressions get bug-scenario-named tests (`progress_regression_test.go`). Network/Telegram/Playwright must never be touched by tests.
- When a permanent feature/API change ships: update `config.example.toml` if config, `docs/` if user-facing, `plugins/README.md` if plugin contract, and i18n YAML + `go generate` for new strings.
