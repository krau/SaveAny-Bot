---
title: "Configuration Guide"
---

# Configuration Guide

SaveAnyBot uses the toml format for its configuration files. You can learn more about toml syntax on the [TOML official website](https://toml.io/).

SaveAnyBot needs to read a `config.toml` file in the working directory as its configuration file. If this file is missing, a default file will be created, and the bot will attempt to load configuration from environment variables.

Here is an example of a minimal configuration file:

```toml
[telegram]
token = "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ"

[[users]]
# telegram user id
id = 777000
blacklist = true

[[storages]]
name = "Local Storage"
type = "local"
enable = true
base_path = "./downloads"
```

## Detailed Configuration

### Global Configuration

- `lang`: The language used by the Bot, default is `zh-CN` (Simplified Chinese). `en` is used for English.
- `stream`: Whether to enable Stream mode, default is `false`. When enabled, the Bot will stream files directly to storage endpoints (if supported), without downloading them locally.
{{< hint warning >}}
Stream mode is very useful for deployment environments with limited disk space, but it also has some drawbacks:
<br />
<ul>
<li>Cannot use multi-threading to download files from Telegram, resulting in slower speeds.</li>
<li>Higher task failure rate when the network is unstable.</li>
<li>Cannot process files in the middle layer, such as automatic file type identification.</li>
<li>Not supported by all storage endpoints; unsupported endpoints may downgrade to normal mode or fail to upload.</li>
</ul>
{{< /hint >}}
- `workers`: Number of tasks to process simultaneously, default is 3.
- `threads`: Number of threads used when downloading files, default is 4. Only effective when Stream mode is not enabled.
- `retry`: Number of retries when a task fails, default is 3.
- `proxy`: Global proxy configuration. After setting this, all network connections inside the program will try to use this proxy. Optional.

```toml
lang = "en"
stream = false
workers = 3
threads = 4
retry = 3
proxy = "socks5://127.0.0.1:7890"
```

### Telegram Configuration

- `token`: Your Telegram Bot Token, which can be obtained by creating a Bot through [BotFather](https://t.me/botfather).
- `app_id`, `app_hash`: Telegram API ID & Hash, obtained by creating an application at [Telegram API](https://my.telegram.org/apps). Default values will be used if not provided.
- `flood_retry`: Number of retries for flood control, default is 5.
- `rpc_retry`: Number of retries for RPC requests, default is 5.
- `proxy`: Proxy configuration, optional.
  - `enable`: Whether to enable the proxy.
  - `url`: Proxy address
- `userbot`: Userbot configuration, optional.
  - `enable`: Enable userbot integration. Requires logging in with a user account; you should use your own API ID & Hash when enabling this.
  - `session`: Path to the userbot session file, default is `data/usersession.db`.

{{< hint warning >}}
After enabling userbot integration, the bot can download files from private channels and groups, but there is an unavoidable risk of the account being banned.
<br />
On the first start after enabling userbot, you need to input phone number, 2FA and verification code in the terminal.
<br />
If you deploy with Docker, please run the container with `-it` for an interactive environment, then perform the login.
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

### Storage Endpoints List

The storage endpoints list is used to define the storage locations supported by the Bot. Each storage endpoint needs to specify a name, type, and related configuration, using the double bracket syntax `[[storages]]`.

Each storage endpoint requires at least the following fields:

- `name`: Storage endpoint name, used for identification in the Bot, must be unique.
- `enable`: Whether to enable this storage endpoint, default is `true`.
- `type`: Storage endpoint type, currently supports the following types:
  - `local`: Local disk
  - `alist`: Alist
  - `webdav`: WebDAV
  - `s3`: aws S3 and other S3 compatible services
  - `telegram`: Upload to Telegram

Example, this is a configuration that includes local storage and webdav storage:

```toml
[[storages]]
name = "Local Storage"
type = "local"
enable = true
# Custom configuration for local type storage
base_path = "./downloads"

[[storages]]
name = "WebDAV"
type = "webdav"
enable = true
# Custom configuration for webdav type storage
url = "https://example.com/webdav"
base_path = "/path/to/webdav"
username = "your_username"
password = "your_password"
```

For custom configuration items for all storage endpoints, see [Storage Configuration](./storages)

### User List

The user list is used to define access control for storage endpoints. Each user needs to specify a Telegram User ID, defined using the double bracket syntax `[[users]]`.

- `id`: The user's Telegram User ID
- `storages`: Filtered list of storage endpoints, defined by storage endpoint names, default is whitelist mode (i.e., only allows access to storage endpoints in the list)
- `blacklist`: Whether to enable blacklist mode, default is `false`. If blacklist mode is enabled, the user is allowed to access only storage endpoints that are **not** in the list.

Example, this is a configuration containing three users: user `123123` can only access local storage, user `456456` can only access storage other than WebDAV, and user `789789` has blacklist mode enabled but no storage endpoints specified, so they can access all storage:

```toml
[[users]]
id = 123123
storages = ["Local Storage"]

[[users]]
id = 456456
storages = ["WebDAV"]
blacklist = true

[[users]]
id = 789789
storages = []
blacklist = true
```

### Events

Event hooks allow you to run custom commands based on task status while the bot is processing tasks. Currently only arbitrary command execution is supported, configured via `[hook.exec]`.

Supported event types:

- `task_before_start`: Before a task starts
- `task_success`: After a task completes successfully
- `task_fail`: After a task fails
- `task_cancel`: After a task is cancelled

The configured value must be a full shell command line. The bot will execute this command when the event occurs. Example:

```toml
[hook.exec]
task_before_start = "echo 'task is about to start'"
task_success = "bash /path/to/success_script.sh"
task_fail = "curl -X POST https://example.com/api/notify -d 'task failed'"
task_cancel = "bash /path/to/cancel_script.sh"
```

### Parsers

Parsers give the bot the ability to handle non-Telegram files, such as downloading files from other websites. Configure them via `[parsers]`.

```toml
[parsers]
plugin_enable = true # Whether to enable parser plugins
plugin_dirs = ["./plugins"] # Plugin directories, can be multiple
```

The above settings only control JavaScript-based parser plugins. The bot also has built-in parsers implemented in Go, which are enabled by default.

### Miscellaneous

```toml
no_clean_cache = false # Whether not to clear the cache folder when exiting
# Temporary download folder configuration
[temp]
base_path = "./cache"
```