# Save Any Bot

把 Telegram 的文件保存到各类存储端. 

> *就像 PikPak Bot 一样*

## 部署

在 [Release](https://github.com/krau/SaveAny-Bot/releases) 页面下载对应平台的二进制文件.

在解压后目录新建 `config.toml` 文件, 参考 [config.toml.example](https://github.com/krau/SaveAny-Bot/blob/main/config.example.toml) 编辑配置文件.

> [!TIP]
> 由于 Telegram 官方 Bot API 的限制, Bot 无法下载大于 20MB 的文件. 你需要部署一个本地的 Telegram Bot API 来解决这个问题, 然后将配置文件中的 telegram.api 改为你自己的 api 地址.
>
> 参考: [telegram-bot-api-compose](https://github.com/krau/telegram-bot-api-compose)

运行:

```bash
chmod +x saveany-bot
./saveany-bot
```

## 使用

向 Bot 发送(转发)文件, 按照提示操作.