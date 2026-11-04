---
title: "Aria2 下载"
weight: 6
---

# Aria2 下载

{{< hint warning >}}
需要在 **设置 → Aria2** 中启用 Aria2 并配置 RPC 连接。
{{< /hint >}}

Aria2 可通过 `/download` 使用。发送链接后，选择 **Aria2**。它支持 HTTP/HTTPS、FTP、BitTorrent 和磁力链接。

```bash
/download <链接1> [链接2] [链接3] ...
```

示例：

```bash
# 下载 HTTP 链接
/download https://example.com/file.zip

# 下载磁力链接
/download magnet:?xt=urn:btih:...

# 下载种子文件
/download https://example.com/file.torrent
```

选择 **Aria2**，然后选择目标存储位置。若只有 Aria2 支持这些链接，则只会提供 Aria2 方式。

在 Web 控制台的 **设置 → Aria2** 中配置：

- `URL`: Aria2 RPC 地址，例如 `http://localhost:6800/jsonrpc`
- `Secret`: 配置了 `rpc-secret` 时填写
- `保留文件`: 转存完成后保留 Aria2 下载的本地文件
