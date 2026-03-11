---
title: "Storage Transfer"
weight: 8
---

# Storage Transfer

Use the `/transfer` command to transfer files directly between different storages without going through Telegram.

```bash
/transfer <source_storage>:/<source_path> [filter]
```

Parameters:

- `source_storage`: Source storage name
- `source_path`: Source path
- `filter`: Optional regex filter to transfer only matching files

Examples:

```bash
# Transfer entire directory
/transfer local1:/downloads

# Transfer files from specified path
/transfer alist1:/media/photos

# Transfer only mp4 files
/transfer webdav1:/videos ".*\.mp4$"

# Transfer image files
/transfer local1:/pictures "(?i)\.(jpg|png|gif)$"
```

The bot will:

1. List all files in the source path
2. Apply the filter (if provided)
3. Display file count and total size
4. Ask you to select the target storage
5. Ask you to select the target directory (if configured for that storage)
6. Start the transfer task

Notes:

- Source storage must support listing and reading
- Target storage must support writing
- Real-time progress is displayed during transfer
- Transfer tasks can be cancelled
