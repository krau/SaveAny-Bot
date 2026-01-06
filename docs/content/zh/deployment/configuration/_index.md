---
title: "配置说明"
---

# 配置说明

SaveAnyBot 的配置文件使用 toml 格式, 你可以在 [TOML 官方网站](https://toml.io/) 上了解更多关于 toml 的语法.

SaveAnyBot 需要读取工作目录下的 `config.toml` 文件作为配置文件, 若缺少该文件则会创建默认文件, 并尝试从环境变量中加载配置.

以下是一个最简的配置文件示例:

```toml
[telegram]
token = "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ"

[[users]]
# telegram user id
id = 777000
blacklist = true

[[storages]]
name = "本机存储"
type = "local"
enable = true
base_path = "./downloads"
```

## 详细配置

### 全局配置

- `stream`: 是否启用 Stream 模式, 默认为 `false`. 启用后 Bot 将直接将文件流式传输到存储端(若存储端支持), 不需要下载到本地
{{< hint warning >}}
Stream 模式对于磁盘空间有限的部署环境十分有用, 但也有一些弊端:
<br />
<ul>
<li>无法使用多线程从 Telegram 下载文件, 速度较慢.</li>
<li>网络不稳定时, 任务失败率高.</li>
<li>无法在中间层对文件进行处理, 例如自动文件类型识别.</li>
<li>并非支持所有存储端, 不支持的存储端可能会降级为普通模式或无法上传.</li>
</ul>
{{< /hint >}}
- `workers`: 同时处理任务数量, 默认为 3
- `threads`: 下载文件时使用的线程数, 默认为 4. 仅在未启用 Stream 模式时生效.
- `retry`: 任务失败时的重试次数, 默认为 3.
- `proxy`: 全局代理配置, 配置后程序内一切网络连接将会尝试使用该代理, 可选.

```toml
stream = false
workers = 3
threads = 4
retry = 3
proxy = "socks5://127.0.0.1:7890"
```

### Telegram 配置

- `token`: 你的 Telegram Bot Token, 可以通过 [BotFather](https://t.me/botfather) 创建 Bot 并获取 Token.
- `app_id`, `app_hash`: Telegram API ID & Hash, 在 [Telegram API](https://my.telegram.org/apps) 创建应用获取, 若不提供则使用默认值.
- `flood_retry`: Flood 控制重试次数, 默认为 5.
- `rpc_retry`: RPC 请求重试次数, 默认为 5.
- `proxy`: 代理配置, 可选.
  - `enable`: 是否启用代理.
  - `url`: 代理地址
- `userbot`: userbot 配置, 可选.
  - `enable`: 启用 userbot 集成, 需要登录用户账号, 此时请务必使用自己的 api id & hash.
  - `session`: userbot 会话文件路径, 默认为 `data/usersession.db`.

{{< hint warning >}}
启用 userbot 集成后, bot 可以下载私密频道和群组的文件, 但具有无法避免的账号被封禁的风险.
<br />
开启 userbot 集成后第一次启动 bot 时需要通过终端交互输入手机号, 2FA 和验证码.
<br />
如果你使用 docker 部署, 请使用 -it 参数为容器提供交互式环境, 然后执行登录操作.
{{< /hint >}}

```toml
[telegram]
token = "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ"
app_id = 1025907
app_hash = "452b0359b988148995f22ff0f4229750"
flood_retry = 5
rpc_retry = 5
[telegram.proxy]
enable = false
url = "socks5://127.0.0.1:7890"
[telegram.userbot]
enable = false
session = "data/usersession.db"
```

### 存储端列表

存储端列表用于定义 Bot 支持的存储位置, 每个存储端需要指定名称、类型和相关配置, 使用双中括号语法 `[[storages]]` 定义.

每一个存储端至少需要以下字段:

- `name`: 存储端名称, 用于在 Bot 中识别, 需要唯一
- `enable`: 是否启用该存储端, 默认为 `true`
- `type`: 存储端类型, 目前支持以下类型:
  - `local`: 本地磁盘
  - `alist`: Alist
  - `webdav`: WebDAV
  - `s3`: aws S3 及其他兼容 S3 的服务
  - `telegram`: 上传到 Telegram

示例, 这是一个包含本地存储和 webdav 存储的配置:

```toml
[[storages]]
name = "本地存储"
type = "local"
enable = true
# 以下是 local 类型存储的自定义配置
base_path = "./downloads"

[[storages]]
name = "WebDAV"
type = "webdav"
enable = true
# 以下是 webdav 类型存储的自定义配置
url = "https://example.com/webdav"
base_path = "/path/to/webdav"
username = "your_username"
password = "your_password"
```

所有存储端的自定义配置项可查看 [存储端配置](./storages) 

### 用户列表

用户列表用于定义对存储端的访问控制, 每个用户需要指定 Telegram 上的用户 ID, 使用双中括号语法 `[[users]]` 定义.

- `id`: 用户的 Telegram User ID
- `storages`: 过滤的存储端列表, 使用存储端名称定义, 默认为白名单模式 (即只允许访问列表中的存储端)
- `blacklist`: 是否启用黑名单模式, 默认为 `false`. 若启用黑名单模式, 则仅允许访问**没有**在列表中的存储端.

示例, 这是一个包含三个用户的配置, 用户 `123123` 只能访问本地存储, 用户 `456456` 只能访问除 WebDAV 以外的存储, 用户 `789789` 启用黑名单模式但没有指定存储端, 因此可以访问所有存储:

```toml
[[users]]
id = 123123
storages = ["本地存储"]

[[users]]
id = 456456
storages = ["WebDAV"]
blacklist = true

[[users]]
id = 789789
storages = []
blacklist = true
```

### 事件触发

事件触发提供了在 Bot 处理任务时根据任务状态执行自定义操作的能力, 目前仅支持任意命令执行. 使用 `[hook.exec]` 配置.

目前具有以下几种事件类型:

- `task_before_start`: 任务即将开始前
- `task_success`: 任务成功完成后
- `task_fail`: 任务失败后
- `task_cancel`: 任务被取消后

提供的配置值需要为完整的命令行命令, Bot 会在事件发生时执行该命令. 示例:

```toml
[hook.exec]
task_before_start = "echo '任务即将开始'"
task_success = "bash /path/to/success_script.sh"
task_fail = "curl -X POST https://example.com/api/notify -d '任务失败'"
task_cancel = "bash /path/to/cancel_script.sh"
```

### 解析器

解析器为 Bot 提供了处理非 Telegram 文件的能力, 例如从其他网站下载文件. 使用 `[parsers]` 配置.

```toml
[parsers]
plugin_enable = true # 是否启用解析器插件
plugin_dirs = ["./plugins"] # 插件目录, 可以是多个目录
```

上述两个配置项只用于控制以 JavaScript 编写的解析器插件, Bot 还有内置的使用 Go 实现的解析器, 目前默认开启.

### 杂项

```toml
no_clean_cache = false # 是否在退出时不清空缓存文件夹
# 临时下载文件夹配置
[temp]
base_path = "./cache"
```