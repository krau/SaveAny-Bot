---
title: "监听聊天"
weight: 4
---

# 监听聊天

{{< hint warning >}}
该功能需开启 UserBot 集成.
{{< /hint >}}

监听指定聊天的消息, 并自动保存到默认存储中, 遵从存储规则, 并且可以设置过滤器来只保存匹配的消息.

监听聊天:

```
/watch <chat_id/username> [filter] 
```

取消监听:

```
/unwatch <chat_id/username>
```

过滤器类型:

## msgre

正则匹配消息文本, 例如:

```
/watch 12345678 msgre:.*hello.*
```

这将会监听 ID 为 12345678 的聊天, 并且只保存消息文本中包含 "hello" 的消息.

## 任务通知

监听任务的通知默认关闭. 开启后, 机器人从被监听聊天中保存的每条消息都会在此处收到通知, 并随下载进度更新, 最终显示保存结果:

```
/watch notify on
```

使用 `/watch notify off` 关闭, 使用 `/watch notify` 查询当前状态.

通知由机器人发送到你与它的私聊中.
