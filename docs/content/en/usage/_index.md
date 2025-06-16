---
title: "Usage"
weight: 10
---

# Usage

## File Transfer

The bot accepts two types of messages: files and links.

Supported links:

1. Telegram message links, for example: `https://t.me/acherkrau/1097`. **Even if the channel prohibits forwarding and saving, the bot can still download its files.**
2. Telegra.ph article links, the bot will download all images within.

## Silent Mode

Use the `/silent` command to toggle silent mode.

By default, silent mode is off, and the bot will ask you for the save location of each file.

When silent mode is enabled, the bot will save files directly to the default location without confirmation.

Before enabling silent mode, you need to set the default save location using the `/storage` command.


## Storage Rules

Allows you to set some redirection rules for the bot when uploading files to storage, for automatic organization of saved files.

See: <a href="https://github.com/krau/SaveAny-Bot/issues/28" target="_blank">#28</a>

Currently supported rule types:

1. FILENAME-REGEX
2. MESSAGE-REGEX

Basic syntax for adding rules:

"Rule Type Rule Content Storage Name Path"

Pay attention to the use of spaces; the bot can only parse correctly formatted syntax. Below is an example of a valid rule command:

```
/rule add FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /videos
```

Additionally, if "CHOSEN" is used as the storage name in the rule, it means the file will be stored in the path of the storage selected via button click.

Rule descriptions:

### FILENAME-REGEX

Matches based on filename regex. The rule content must be a valid regular expression, such as:

```
FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /videos
```

This means files with extensions mp4, mkv, ts, avi, flv will be saved to the /videos directory in the storage named MyAlist (also affected by the `base_path` in the configuration file).

### MESSAGE-REGEX

Similar to the above, but matches based on the text content of the message itself.