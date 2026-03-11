---
title: "存储间传输"
weight: 8
---

# 存储间传输

使用 `/transfer` 命令可以在不同存储之间直接传输文件, 无需经过 Telegram.

```bash
/transfer <source_storage>:/<source_path> [filter]
```

参数说明:

- `source_storage`: 源存储名称
- `source_path`: 源路径
- `filter`: 可选的正则表达式过滤器, 只传输匹配的文件

示例:

```bash
# 传输整个目录
/transfer local1:/downloads

# 传输指定路径的文件
/transfer alist1:/media/photos

# 只传输 mp4 文件
/transfer webdav1:/videos ".*\.mp4$"

# 传输图片文件
/transfer local1:/pictures "(?i)\.(jpg|png|gif)$"
```

Bot 会:

1. 列出源路径下的所有文件
2. 应用过滤器 (如果提供)
3. 显示文件数量和总大小
4. 让你选择目标存储
5. 让你选择目标目录 (如果该存储配置了目录)
6. 开始传输任务

注意:

- 源存储必须支持列举和读取功能
- 目标存储必须支持写入功能
- 传输过程显示实时进度
- 支持取消正在进行的传输任务
