我真的服了😇, 目前没有什么好的 telegram mtproto sdk, 因此这个项目在运行时会产生很多因上游依赖的 bug 的终止, 能跑就是赢.

# Save Any Bot

把 Telegram 的文件保存到各类存储端.

> _就像 PikPak Bot 一样_

## 部署

在 [Release](https://github.com/krau/SaveAny-Bot/releases) 页面下载对应平台的二进制文件.

在解压后目录新建 `config.toml` 文件, 参考 [config.toml.example](https://github.com/krau/SaveAny-Bot/blob/main/config.example.toml) 编辑配置文件.

运行:

```bash
chmod +x saveany-bot
./saveany-bot
```

## 使用

向 Bot 发送(转发)文件, 按照提示操作.

## Bot API 版本 (v0.3.0 前)

> Bot API 版本自身不需要 API_ID 和 API_HASH, 但是部署 Telegram Bot API 服务器仍然需要.

由于 Telegram 官方 Bot API 的限制, Bot 无法下载大于 20MB 的文件. 你需要部署一个本地的 Telegram Bot API 来解决这个问题, 然后在配置文件改为你自己的 api 地址

```toml
[telegram]
api = "http://localhost:8081"
```

参考: [telegram-bot-api-compose](https://github.com/krau/telegram-bot-api-compose)
