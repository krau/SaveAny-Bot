---
title: "Aria2 下载"
weight: 6
---

# Aria2 下载

{{< hint warning >}}
该功能需要在 **设置 → Aria2** 中启用并配置 RPC 连接.
{{< /hint >}}

使用 `/aria2dl` 命令可以通过 Aria2 下载管理器下载文件, 支持 HTTP/HTTPS、FTP、BitTorrent 等多种协议.

```bash
/aria2dl <uri1> [uri2] [uri3] ...
```

示例:

```bash
# 下载 HTTP 链接
/aria2dl https://example.com/file.zip

# 下载磁力链接
/aria2dl magnet:?xt=urn:btih:...

# 下载种子文件 (需要先上传 .torrent 文件)
/aria2dl https://example.com/file.torrent
```

配置 Aria2:

在 Web 控制台的 **设置 → Aria2** 中启用，并填写:

- `URL`: Aria2 RPC 地址，例如 `http://localhost:6800/jsonrpc`
- `Secret`: 配置了 `rpc-secret` 时填写
- `保留文件`: 转存完成后是否保留 Aria2 下载的本地文件
