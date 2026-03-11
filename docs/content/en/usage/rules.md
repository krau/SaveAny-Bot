---
title: "Storage Rules"
weight: 3
---

# Storage Rules

Storage rules allow you to define redirection rules when the bot uploads files to storage, so that saved files are automatically organized.

See: <a href="https://github.com/krau/SaveAny-Bot/issues/28" target="_blank">#28</a>

Currently supported rule types:

1. FILENAME-REGEX
2. MESSAGE-REGEX
3. IS-ALBUM

Basic syntax for adding rules:

"RuleType RuleContent StorageName Path"

Pay attention to spaces; the bot can only parse correctly formatted syntax. Below is an example of a valid rule command:

```
/rule add FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /videos
```

In addition, if `CHOSEN` is used as the storage name in the rule, it means files will be stored under the path of the storage you selected by clicking the inline button.

Rule types:

## FILENAME-REGEX

Matches based on filename regex. The rule content must be a valid regular expression, such as:

```
FILENAME-REGEX (?i)\.(mp4|mkv|ts|avi|flv)$ MyAlist /videos
```

This means files with extensions mp4, mkv, ts, avi, flv will be saved to the `/videos` directory in the storage named `MyAlist` (also affected by the `base_path` in the configuration file).

## MESSAGE-REGEX

Similar to the above, but matches based on the text content of the message itself.

## IS-ALBUM

Matches album messages (media groups). Rule content can only be `true` or `false`.

If the path in the rule uses `NEW-FOR-ALBUM`, the bot will create a new folder for each media group and store all files of that group there. See: https://github.com/krau/SaveAny-Bot/issues/87

For example:

```
IS-ALBUM true MyWebdav NEW-FOR-ALBUM
```

This will save media-group messages to the storage named `MyWebdav`, creating a new folder (generated from the first file) for each album.
