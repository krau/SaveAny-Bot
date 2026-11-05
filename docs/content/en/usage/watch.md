---
title: "Watch Chats"
weight: 4
---

# Watch Chats

{{< hint warning >}}
This feature requires enabling UserBot integration.
{{< /hint >}}

You can watch messages in a specific chat and automatically save them to the default storage, following storage rules. You can also add filters so that only matching messages are saved.

Send `/watch` to manage watched chats:

- **Add watch** — shows the syntax for adding a chat
- **Watched chats** — lists watched chats and lets you stop each one
- **Notifications** — toggles task notifications for all watched chats
- **Sync peers** — imports the Userbot dialog list into the peer cache

To start watching a chat, send:

```
/watch <chat_id/username> [filter]
```

To stop watching, open **Watched chats** from `/watch` and select the chat.

Filter types:

## msgre

Regex-match the message text. For example:

```
/watch -1002229835658 msgre:.*hello.*
```

This will watch the channel with ID `-1002229835658`, and only save messages whose text contains `hello`. Use the Bot API form: positive IDs for users, negative IDs for basic groups, and `-100...` for supergroups and channels.

## Task notifications

Task notifications are disabled by default for every user. Enable them with `/watch` → **Notifications** to get a message for every message the bot picks up from a watched chat, updated with the download progress and the final result. Disable them the same way.

Notifications are sent by the bot to your private chat with it. The toggle is per user, not per watched chat.
