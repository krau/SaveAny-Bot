---
title: "存储端配置"
---

# 存储端配置

在 Web 控制台的 **存储** 页面添加存储时填写以下字段。

存储名称可以包含空格、中文和符号（最多 64 个字符，不能包含控制字符）。在 `/dir`、`/transfer` 等 Bot 命令中使用含空格的名称时请加引号。之后可以直接在同一页面重命名；用户默认存储、目录、规则和存储过滤使用稳定的存储 ID，不受名称变化影响。

## Alist

`type=alist`

不支持 Stream 模式.

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

如果你使用的是第三方的兼容 S3 的服务, 一般使用的是路径风格的 URL. 而 AWS S3 则通常使用虚拟主机风格的 URL. 详情请参考你所使用的 S3 兼容服务的文档.

## Telegram

`type=telegram`

不支持 Stream 模式.

```text
# Telegram 聊天 ID, Bot 将把文件发送到这个聊天
chat_id = "123456789"
# 是否强制使用文件方式发送, 默认为 false
force_file = false
# 是否跳过大文件, 默认为 false. 如果启用, 超过 Telegram 限制的文件将不会上传.
skip_large = false
# 超限视频是否使用 FFmpeg 无损分割成一个媒体组（最多 10 个）内可独立播放的小视频；失败或需要超过 10 段时回退到 ZIP 分卷.
split_large_video = false
# 分卷大小, 单位 MB. 设为 0 时使用实际上传账号的限制：Bot/普通用户为 2000 MB，Premium 用户为 4000 MB.
# 超过该大小的文件将被分割成多个部分上传；非视频使用 ZIP 格式.
# 当 skip_large 启用时, 该选项无效.
split_size_mb = 0
```

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

首先需要配置 rclone 远程, 运行 `rclone config` 命令进行交互式配置, 或直接编辑 `rclone.conf` 文件.

rclone 支持多种云存储服务, 包括但不限于:
- Google Drive
- Dropbox
- OneDrive
- Amazon S3 及兼容服务
- SFTP
- FTP
- 更多服务请参考 [rclone 官方文档](https://rclone.org/overview/)

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
