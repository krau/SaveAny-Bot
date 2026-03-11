---
title: "Watch Chats"
weight: 4
---

# Watch Chats

{{< hint warning >}}
This feature requires enabling UserBot integration.
{{< /hint >}}

You can watch messages in a specific chat and automatically save them to the default storage, following storage rules. You can also add filters so that only matching messages are saved.

Watch a chat:

```
/watch <chat_id/username> [filter]
```

Stop watching:

```
/unwatch <chat_id/username>
```

Filter types:

## msgre

Regex-match the message text. For example:

```
/watch 12345678 msgre:.*hello.*
```

This will watch the chat with ID `12345678`, and only save messages whose text contains `hello`.
