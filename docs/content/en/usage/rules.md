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

You can also toggle whether rules are applied with `/rule switch`. When rule mode is off, all files go to the default storage.

## Preset Rules

Manually writing regex rules for common file types is tedious, so the bot ships a built-in set of preset categories (video, image, audio, document, archive) that you can import in one command:

```
/rule preset <storage> [base_path]
```

Parameters:

- `storage`: Target storage name (must exist and be accessible to you)
- `base_path`: Optional. Each preset category's subdirectory is created under this path. If omitted, the default category directory names are used directly.

Examples:

```
# Import preset rules into "MyAlist" with the default directory layout
/rule preset MyAlist

# Import preset rules with a custom base path "downloads/sorted"
/rule preset MyAlist downloads/sorted
```

This will create `FILENAME-REGEX` rules for each category, routing matched files to the corresponding subdirectory under `base_path`:

| Category | Matched extensions | Default directory |
|---|---|---|
| video | mp4, mkv, ts, avi, flv, mov, webm, wmv, rmvb, m2ts | `视频` |
| image | jpg, jpeg, png, gif, webp, bmp | `图片` |
| audio | mp3, flac, wav, aac, m4a, ogg | `音频` |
| document | pdf, doc, docx, xls, xlsx, ppt, pptx, txt, md, csv, epub, mobi, azw3, chm | `文档` |
| archive | zip, rar, 7z, tar, gz, bz2, xz, ... | `压缩包` |

{{< hint info >}}
Preset rules are regular `FILENAME-REGEX` rules once imported. You can view, edit, or delete them individually with `/rule` and `/rule del <id>` like any other rule.
{{< /hint >}}

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
