---
title: "Aria2 Downloads"
weight: 6
---

# Aria2 Downloads

{{< hint warning >}}
This feature requires enabling Aria2 under **Settings → Aria2** and configuring the RPC connection.
{{< /hint >}}

Aria2 downloads are available through `/download`. Send links, then choose **Aria2**. It supports HTTP/HTTPS, FTP, BitTorrent, and magnet links.

```bash
/download <uri1> [uri2] [uri3] ...
```

Examples:

```bash
# Download an HTTP link
/download https://example.com/file.zip

# Download a magnet link
/download magnet:?xt=urn:btih:...

# Download a torrent file
/download https://example.com/file.torrent
```

Choose **Aria2**, then select the target storage. When only Aria2 supports the supplied links, it is the only engine available.

Configure Aria2 under **Settings → Aria2** in the web console:

- `URL`: Aria2 RPC address, for example `http://localhost:6800/jsonrpc`
- `Secret`: required when `rpc-secret` is configured
- `Keep file`: keep the local file downloaded by Aria2 after the transfer
