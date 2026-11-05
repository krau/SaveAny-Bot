---
title: "存储端配置"
---

# 存储端配置

在 Web 控制台的 **存储** 页面添加存储时填写以下字段。

存储名称支持空格、中文与常用符号（最多 64 字符，不可含控制字符）。在 `/dirs`、`/transfer` 等命令中引用含空格的名称时需加双引号。名称可在 Web 控制台随时更改，用户默认存储、目录及规则均基于内部存储 ID 关联，不受重命名影响。

## Alist

`type=alist`

不支持流式下载（Stream 模式）。

```text
url = "https://alist.example.com" # Alist 的 URL
username = "your_username"  # Alist 的用户名
password = "your_password" # Alist 的密码
base_path = "/path/saveanybot" # Alist 中的基础路径, 所有文件将存储在此路径下
token_exp = 3600 # Alist 访问令牌的自动刷新时间, 单位秒
token = "your_token" 
# Alist 的访问令牌, 可选, 如果不设置则使用用户名和密码进行身份验证. 
# 使用 token 验证时无法自动刷新 token
```

## 本地磁盘

`type=local`

```text
base_path = "./downloads" # 本地存储的基础路径, 所有文件将存储在此路径下
```

## WebDAV
`type=webdav`

```text
url = "https://webdav.example.com" # WebDAV 的 URL
username = "your_username"  # WebDAV
password = "your_password" # WebDAV 的密码
base_path = "/path/to/webdav" # WebDAV 中的基础路径, 所有文件将存储在此路径下
```

## S3

`type=s3`

```text
endpoint = "https://s3.example.com" # S3 的端点, 默认为 aws S3 的端点
region = "us-east-1" # S3 的区域
access_key_id = "your_access_key_id" # S3 的访问密钥 ID
secret_access_key = "your_secret_access_key" # S3 的秘密访问密钥
bucket_name = "your_bucket_name" # S3 的存储桶名称
base_path = "/path/to/s3" # S3 中的基础路径, 所有文件将存储在此路径下
virtual_host = false # 使用虚拟主机风格的 URL, 默认为 false
```

虚拟主机风格的 URL 示例:

```
https://your_bucket_name.s3.example.com/path/to/s3/your_file
```

路径风格(关闭 virtual_host)的 URL 示例:

```
https://s3.example.com/your_bucket_name/path/to/s3/your_file
```

第三方 S3 兼容服务通常采用路径风格 URL，AWS S3 则默认使用虚拟主机风格。具体请参考对应服务文档。

## Telegram

`type=telegram`

不支持流式下载（Stream 模式）。

```text
# Telegram 聊天 ID, Bot 将把文件发送到这个聊天
chat_id = "-1001234567890" # Bot API 聊天 ID（用户为正数，普通群为负数，超级群/频道使用 -100...）
# 是否强制使用文件方式发送, 默认为 false
force_file = false
# 是否跳过大文件, 默认为 false. 如果启用, 超过 Telegram 限制的文件将不会上传.
skip_large = false
# 超限视频是否使用 FFmpeg 分割为单组（最多 10 个）独立播放的小视频；失败或超过 10 段时回退为 ZIP 分卷.
split_large_video = false
# 分卷大小, 单位 MB. 设为 0 时使用实际上传账号的限制：Bot/普通用户为 2000 MB，Premium 用户为 4000 MB.
# 超过该大小的文件将被分割成多个部分上传；非视频使用 ZIP 格式.
# 当 skip_large 启用时, 该选项无效.
split_size_mb = 0
# 没有实时 caption 时，用元数据中的消息文本作为发送 caption
use_metadata_caption = false
# 没有实时分组信息时，用元数据 grouped_id 将文件作为媒体组发送
use_metadata_album = false
```

启用消息元数据后，Telegram 存储会直接消费元数据，不会把 `.meta.json` 作为文件上传。`use_metadata_caption` 在没有实时 caption 时用原消息文本还原 caption；`use_metadata_album` 在没有实时分组信息时用元数据 `grouped_id` 将文件作为媒体组发送。两项默认关闭。媒体组只会在第一个文件上添加 caption，以符合 Telegram 客户端行为。

## Rclone

`type=rclone`

通过 [rclone](https://rclone.org/) 命令行工具支持多种云存储服务. 需要先安装 rclone 并配置好远程存储.

```text
# rclone 配置的远程名称, 可以是任何在 rclone.conf 中配置的远程
remote = "mydrive"
# 在远程存储中的基础路径, 所有文件将存储在此路径下
base_path = "/telegram"
# rclone 配置文件的路径, 可选, 留空使用默认路径 (~/.config/rclone/rclone.conf)
config_path = ""
# 传递给 rclone 命令的额外参数, 可选
flags = ["--transfers", "4", "--checkers", "8"]
```

### 配置 rclone 远程

首先需要配置 rclone 远程，运行 `rclone config` 进行交互式配置，或直接编辑 `rclone.conf` 文件。

支持 Google Drive、Dropbox、OneDrive、S3 兼容存储、SFTP/FTP 等服务，详见 [rclone 官方文档](https://rclone.org/overview/)。

### 使用示例

配置 Google Drive 后, 可以这样配置存储:

```text
[[storages]]
name = "GoogleDrive"
type = "rclone"
enable = true
remote = "gdrive"
base_path = "/SaveAnyBot"
```

如果使用自定义的 rclone 配置文件:

```text
[[storages]]
name = "MyRemote"
type = "rclone"
enable = true
remote = "myremote"
base_path = "/backup"
config_path = "/path/to/rclone.conf"
flags = ["--progress"]
```
