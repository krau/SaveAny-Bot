---
title: "Storage Configuration"
---

# Storage Configuration

Add storages on the **Storages** page in the web console and fill in the fields below.

Storage names can contain spaces, Unicode, and symbols (up to 64 characters, excluding control characters). Quote names containing spaces when using bot commands such as `/dirs` or `/transfer`. Rename a storage on the same page at any time; user defaults, directories, rules, and storage filters reference the stable storage ID, so they are unaffected by the name change.

## Alist

`type=alist`

Stream mode is not supported.

```text
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

```text
base_path = "./downloads" # Base path for local storage, all files will be stored under this path
```

## WebDAV
`type=webdav`

```text
url = "https://webdav.example.com" # URL of WebDAV
username = "your_username"  # Username for WebDAV
password = "your_password" # Password for WebDAV
base_path = "/path/to/webdav" # Base path in WebDAV, all files will be stored under this path
```

## S3

`type=s3`

```text
endpoint = "https://s3.example.com" # Endpoint for S3, defaults to AWS S3 endpoint if not set
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

```text
chat_id = "-1001234567890" # Bot API chat ID (positive for users, negative for basic groups, -100... for channels/supergroups)
force_file = false # Force sending as file, default is false
skip_large = false # Skip large files, default is false. If enabled, files exceeding Telegram's limit will not be uploaded.
split_large_video = false # Losslessly split oversized videos into one album of up to 10 independently playable parts. Falls back to ZIP parts on failure or when more than 10 parts are required.
split_size_mb = 0 # Split size in MB. 0 uses the uploader account limit: 2000 MB for bots/regular users and 4000 MB for Premium users. Oversized non-video files use ZIP parts. Ignored when skip_large is true.
```

## Rclone

`type=rclone`

Supports multiple cloud storage services through the [rclone](https://rclone.org/) command-line tool. You need to install rclone and configure remote storage first.

```text
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

```text
[[storages]]
name = "GoogleDrive"
type = "rclone"
enable = true
remote = "gdrive"
base_path = "/SaveAnyBot"
```

If using a custom rclone config file:

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
