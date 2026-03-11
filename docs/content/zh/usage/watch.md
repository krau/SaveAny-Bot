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
