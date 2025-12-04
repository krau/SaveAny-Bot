---
title: "存储端配置"
---

# 存储端配置

请先阅读 [配置说明](../) 了解配置文件的基本格式.

## Alist

`type=alist`

不支持 Stream 模式.

```toml
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

```toml
base_path = "./downloads" # 本地存储的基础路径, 所有文件将存储在此路径下
```

## WebDAV
`type=webdav`

```toml
url = "https://webdav.example.com" # WebDAV 的 URL
username = "your_username"  # WebDAV
password = "your_password" # WebDAV 的密码
base_path = "/path/to/webdav" # WebDAV 中的基础路径, 所有文件将存储在此路径下
```

## S3

`type=s3`

```toml
endpoint = "s3.example.com" # S3 的端点
region = "us-east-1" # S3 的区域
access_key_id = "your_access_key_id" # S3 的访问密钥 ID
secret_access_key = "your_secret_access_key" # S3 的秘密访问密钥
bucket_name = "your_bucket_name" # S3 的存储桶名称
use_ssl = true # 是否使用 SSL, 默认为 true
base_path = "/path/to/s3" # S3 中的基础路径, 所有文件将存储在此路径下
```

## Telegram

`type=telegram`

不支持 Stream 模式.

```toml
chat_id = "123456789" # Telegram 聊天 ID, Bot 将把文件发送到这个聊天
```