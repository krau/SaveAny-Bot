---
title: "Storage Configuration"
---

# Storage Configuration

Please first read the [Configuration Guide](../) to understand the basic format of the configuration file.

## Alist

`type=alist`

Stream mode is not supported.

```toml
url = "https://alist.example.com" # URL of Alist
username = "your_username"  # Username for Alist
password = "your_password" # Password for Alist
base_path = "/path/saveanybot" # Base path in Alist, all files will be stored under this path
token_exp = 3600 # Auto-refresh time for Alist access token, in seconds
token = "your_token" 
# Access token for Alist, optional, if not set, username and password will be used for authentication.
# When using token authentication, the token cannot be automatically refreshed
```

## Local Disk

`type=local`

```toml
base_path = "./downloads" # Base path for local storage, all files will be stored under this path
```

## WebDAV
`type=webdav`

```toml
url = "https://webdav.example.com" # URL of WebDAV
username = "your_username"  # Username for WebDAV
password = "your_password" # Password for WebDAV
base_path = "/path/to/webdav" # Base path in WebDAV, all files will be stored under this path
```

## S3

`type=s3`

```toml
endpoint = "s3.example.com" # Endpoint for S3, defaults to AWS S3 endpoint if not set
region = "us-east-1" # Region for S3
access_key_id = "your_access_key_id" # Access key ID for S3
secret_access_key = "your_secret_access_key" # Secret access key for S3
bucket_name = "your_bucket_name" # Bucket name for S3
base_path = "/path/to/s3" # Base path in S3, all files will be stored under this path
virtual_host = false # Use virtual-host style URL, default is false
```

Example of virtual-host-style URL:

```
https://your_bucket_name.s3.example.com/path/to/s3/your_file
```

Example of path-style URL (when `virtual_host` is false):

```
https://s3.example.com/your_bucket_name/path/to/s3/your_file
```

If you are using a third-party S3-compatible service, it usually uses path-style URLs. AWS S3 typically uses virtual-host-style URLs. Please refer to your S3-compatible service documentation for details.

## Telegram

`type=telegram`

Stream mode is not supported.

```toml
chat_id = "123456789" # Telegram chat ID, the bot will send files to this chat
force_file = false # Force sending as file, default is false
skip_large = false # Skip large files, default is false. If enabled, files exceeding Telegram's limit will not be uploaded.
spilt_size_mb = 2000 # Split size in MB, default is 2000 MB (2 GB). Files larger than this will be split into multiple parts (zip format). Ignored when skip_large is true.
```

## Rclone

`type=rclone`

Supports multiple cloud storage services through the [rclone](https://rclone.org/) command-line tool. You need to install rclone and configure remote storage first.

```toml
# Remote name configured in rclone, can be any remote defined in rclone.conf
remote = "mydrive"
# Base path in the remote storage, all files will be stored under this path
base_path = "/telegram"
# Path to rclone config file, optional, leave empty to use default path (~/.config/rclone/rclone.conf)
config_path = ""
# Additional flags to pass to rclone commands, optional
flags = ["--transfers", "4", "--checkers", "8"]
```

### Configuring rclone Remote

First, you need to configure an rclone remote. Run `rclone config` for interactive configuration, or directly edit the `rclone.conf` file.

rclone supports many cloud storage services, including but not limited to:
- Google Drive
- Dropbox
- OneDrive
- Amazon S3 and compatible services
- SFTP
- FTP
- For more services, please refer to the [rclone official documentation](https://rclone.org/overview/)

### Usage Examples

After configuring Google Drive, you can configure the storage like this:

```toml
[[storages]]
name = "GoogleDrive"
type = "rclone"
enable = true
remote = "gdrive"
base_path = "/SaveAnyBot"
```

If using a custom rclone config file:

```toml
[[storages]]
name = "MyRemote"
type = "rclone"
enable = true
remote = "myremote"
base_path = "/backup"
config_path = "/path/to/rclone.conf"
flags = ["--progress"]
```