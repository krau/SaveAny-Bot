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
endpoint = "s3.example.com" # Endpoint for S3
region = "us-east-1" # Region for S3
access_key_id = "your_access_key_id" # Access key ID for S3
secret_access_key = "your_secret_access_key" # Secret access key for S3
bucket_name = "your_bucket_name" # Bucket name for S3
use_ssl = true # Whether to use SSL, default is true
base_path = "/path/to/s3" # Base path in S3, all files will be stored under this path
```

## Telegram

`type=telegram`

Stream mode is not supported.

```toml
chat_id = "123456789" # Telegram chat ID, the Bot will send files to this chat
```