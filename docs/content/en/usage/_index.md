---
title: "Usage"
weight: 10
---

# Usage

This page introduces some of Save Any Bot's features and basic usage. If you can't find what you need here, please also see the [Configuration Guide](../deployment/configuration) or ask in GitHub [Discussions](https://github.com/krau/SaveAny-Bot/discussions).

## Bot Commands

| Command | Purpose |
|---|---|
| `/start` | Open help and the command list |
| `/save` | Save a replied media message, or batch-save a message range |
| `/download` | Download links using Direct, Aria2, or Video |
| `/transfer` | Transfer files between storages |
| `/tasks` | Inspect and manage running, queued, and failed tasks |
| `/config` | User settings: filename and conflict strategy, silent mode, default storage |
| `/template` | Set the custom filename template |
| `/dirs` | Manage storage directories |
| `/rules` | Manage storage rules |
| `/watch` | Watch chats and manage notifications/peer sync (UserBot) |
| `/parser` | Manage parser plugins |
| `/update` | Check for updates |


## File Transfer

To use the bot's Telegram file saving feature, you need to send or forward the following types of messages to the bot:

1. File or media messages, such as images, videos, documents, etc.
2. Telegram message links, for example: `https://t.me/acherkrau/1097`. **Even if the channel prohibits forwarding and saving, the bot can still download its files.**
3. Telegra.ph article links. The bot will download all images in the article.

## Group Chats

The bot works in basic groups and supergroups. Broadcast channels are not supported.

- In groups the bot only reacts to explicit commands and button callbacks; ordinary media and link messages are not saved automatically.
- To save a file from the group, reply to it with `/save`, or use `/save@YourBot` in groups. Other commands such as `/download` work as usual.
- Only whitelisted users can use the bot. Messages from other members are ignored silently, and their button clicks receive a permission alert.
- Messages sent by anonymous administrators in supergroups have no identifiable user and are ignored.
- With the default Privacy Mode, the bot only receives commands addressed to it, mentions, and replies to its own messages. To let it receive all group messages, use `/setprivacy` in [@BotFather](https://t.me/BotFather) to disable Privacy Mode and re-add the bot to the group. Even then, the bot only reacts to commands and buttons in groups.