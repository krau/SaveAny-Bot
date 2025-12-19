# SaveAny-Bot AI 协作说明

本项目是一个将 Telegram 文件/消息转存到多种存储端的 Bot，主要用 Go 实现，CLI 入口在 `cmd/`，核心逻辑分布在 `client/`、`core/`、`config/`、`database/`、`storage/` 等目录。下面是针对本仓库的专用约定，供 AI 编码助手参考。

## 总体架构与入口
- **CLI 入口**：
  - 二进制入口：`main.go` 调用 `cmd.Execute(ctx)`。
  - 根命令：`cmd/root.go` 使用 `cobra` 定义 `saveany-bot`，`Run` 实现在 `cmd/run.go`。
- **应用启动流程（非常重要）**：
  - `cmd/run.go::Run` 中按顺序完成：读取配置 `config.Init` → 初始化缓存 `common/cache` → 初始化 i18n `common/i18n` → 初始化数据库 `database.Init` → 加载存储 `storage.LoadStorages` → 加载解析器插件 `parsers.LoadPlugins` → （可选）Userbot 登录 → 启动 Telegram Bot `client/bot.Init` → 启动核心任务队列消费 `core.Run`。
  - 添加新的初始化步骤时，请遵循该顺序并放在 `initAll` 中，而不是分散在各处。

## 配置与约定
- **配置系统**：
  - 使用 `viper` 读取 `config.toml`，核心结构体定义在 `config/viper.go::Config`。
  - 默认值在 `config.Init` 中通过 `viper.SetDefault` 定义，如 `workers`、`retry`、`telegram.*`、`db.*` 等。
  - `Config.C()` 返回的是全局配置副本（值类型），不要在返回值上修改字段；如果需要修改配置流程，应在 `config.Init` 内或通过 `viper` 进行。
  - 存储配置通过 `config/storage/factory.go::LoadStorageConfigs` 加载并校验，新增存储类型需：
    - 在 `pkg/enums/storage` 中增加枚举。
    - 在 `config/storage/` 下新增具体 `StorageConfig` 实现并实现 `Validate`。
    - 在 `storageFactories` 映射中注册工厂方法。
- **环境变量**：
  - 所有配置键会被 `SAVEANY_` 前缀的环境变量覆盖，`.` 会被 `_` 替换（`SAVEANY_TELEGRAM_APP_ID` 等）。

## Telegram 客户端与中间件
- **Bot 客户端**：
  - 入口在 `client/bot/bot.go::Init`，使用 `gotgproto.NewClient` 创建，Session 使用 `database.GetDialect(config.C().DB.Session)`，错误处理通过 `ErrorHandler` 回调完成。
  - Handlers 注册集中在 `client/bot/handlers` 目录，`handlers.Register` 负责统一挂载；新增命令/消息处理逻辑时优先在该目录按功能拆分文件，实现后在 `handlers.Register` 中注册。
  - Bot 命令列表依赖 `handlers.CommandHandlers` 来自动注册到 Telegram；新增命令时务必更新该切片，以保持 `/help` 与 Bot 命令列表一致。
- **中间件**：
  - 通用中间件位于 `client/middleware/`，包含 floodwait、防崩溃、重试等；`middleware.NewDefaultMiddlewares` 在 `client/bot/bot.go` 中统一挂载。
  - 新增跨所有更新生效的行为（如日志、统计）时，应优先实现为中间件。

## 核心任务与队列
- **任务接口与队列**：
  - 核心接口：`core/core.go::Executable`，包含 `Type() TaskType`、`Title()`、`TaskID()`、`Execute(ctx)`。
  - 任务队列：`pkg/queue.TaskQueue[Executable]`，由 `core.Run` 使用；`Workers` 数量来自配置 `config.C().Workers`。
  - 任务类型与实现示例位于 `core/tasks/**`，例如文件任务、Telegraph 任务等；新增任务类型应放在对应子目录并实现 `Executable` 接口，然后通过 `core.AddTask` 入队。
- **生命周期 Hook**：
  - `core.worker` 在执行任务前后会根据 `config.C().Hook.Exec` 调用外部命令（`TaskBeforeStart` / `TaskSuccess` / `TaskFail` / `TaskCancel`）。
  - 修改任务执行流程时需保留这些 Hook 调用，以免破坏用户已有集成。

## 数据库与持久化
- **数据库初始化**：
  - `database.Init` 使用配置 `config.C().DB.Path` 创建并连接 SQLite，使用 `GetDialect` 抽象驱动（见 `database/driver_*.go`）。
  - Migration 通过 `db.AutoMigrate(&User{}, &Dir{}, &Rule{}, &WatchChat{})` 完成，模型定义在 `database/*.go` 中。
- **用户同步约定**：
  - `database.syncUsers` 会根据 `config.C().Users` 同步数据库用户表：在配置中新增/删除用户会自动在 DB 中创建/删除对应记录。
  - 开发涉及用户表逻辑时，请考虑该同步行为，避免在其他地方直接创建/删除用户记录而与配置冲突。

## 存储后端
- **存储抽象**：
  - 抽象接口在 `config/storage/types.go` 与 `storage/` 顶层（以及子目录）中；`config/storage/*.go` 处理配置解析，`storage/*` 处理真正的上传/下载实现。
  - 现有实现包括 `local`、`alist`、`s3/minio`、`webdav`、`telegram` 等，每个后端都有对应子目录和配置结构体。
- **新增存储实现的推荐路径**：
  - 在 `config/storage/` 下添加配置结构体 + `Validate`。
  - 在 `storage/` 下添加具体实现（例如 `storage/foo/`）。
  - 在 `pkg/enums/storage` 与 `storageFactories` 中注册，并确保 `storages` 配置示例被更新（`config.example.toml` / 文档）。

## 解析器插件（JS）
- **插件运行时**：
  - 解析器接口和插件文档在 `plugins/README.md`，Go 端入口为 `parsers/` 目录，使用 `goja` 与 `playwright-go`。
  - 插件通过 `registerParser({ metadata, canHandle, parse })` 注册，使用 `ghttp`/`playwright` 进行 HTTP/浏览器请求。
- **与核心交互约定**：
  - 插件 `parse` 返回的 `Item`/`Resource` 会被转化为内部任务（通常是下载/转存任务）并进入 `core` 队列。
  - 修改 `Item`/`Resource` 结构或解析逻辑时，要确保保持向后兼容，或在 `plugins/README.md` 中同步更新字段说明和示例。

## i18n 与日志
- **国际化**：
  - 所有用户可见字符串（尤其是错误与提示）应使用 `common/i18n`：`i18n.T(i18nk.SomeKey, map[string]any{"Name": name})`。
  - 语言文件位于 `common/i18n/locale/`，`go:generate` 指令在 `main.go` 中生成 `i18nk/keys.go`；新增文案时需：添加到 YAML、运行 `go generate ./...`、再在代码中引用新 key。
- **日志**：
  - 使用 `github.com/charmbracelet/log`，在 `cmd/run.go::Run` 中通过 `log.WithContext` 将 logger 注入 `context.Context`；后续代码优先通过 `log.FromContext(ctx)` 获取 logger。
  - 编写新代码时，如已有 `ctx`，请使用 `log.FromContext(ctx)` 而不是全局 logger。

## 开发与运行
- **本地运行**：
  - 直接运行：`go run ./cmd`（`cmd/root.go` + `cmd/run.go`）。
  - 或通过 Docker：参见根目录 `README.md` 中的 `docker run ...` 示例及 `docker-compose.yml`。
- **代码生成与文档**：
  - i18n key 生成：`go generate ./...` 会执行 `main.go` 顶部的 `//go:generate`，使用 `cmd/geni18n/main.go` 生成 `common/i18n/i18nk/keys.go`。
  - 文档站点在 `docs/`（Hugo），通常不需要在核心代码改动时同步修改，除非涉及文档内容。

---

如果你在实现某个功能时发现以上规则不够具体（例如某类任务在 `core/tasks` 中到底如何落地，或某个存储/解析器的边界不清晰），请在对应章节下扩展更精确的说明，并在 PR 描述中标注。也欢迎你告诉我有哪些部分需要补充或澄清，我可以进一步细化。