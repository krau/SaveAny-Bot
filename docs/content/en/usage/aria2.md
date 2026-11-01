---
title: "Aria2 Download"
weight: 6
---

# Aria2 Download

{{< hint warning >}}
This feature requires enabling Aria2 under **Settings → Aria2** and configuring the RPC connection.
{{< /hint >}}

Use the `/aria2dl` command to download files via the Aria2 download manager, supporting HTTP/HTTPS, FTP, BitTorrent, and other protocols.

```bash
/aria2dl <uri1> [uri2] [uri3] ...
```

Examples:

```bash
# Download HTTP link
/aria2dl https://example.com/file.zip

# Download magnet link
/aria2dl magnet:?xt=urn:btih:...

# Download torrent file (need to upload .torrent file first)
/aria2dl https://example.com/file.torrent
```

Configure Aria2:

Enable it under **Settings → Aria2** in the web console and fill in:

- `URL`: Aria2 RPC address, for example `http://localhost:6800/jsonrpc`
- `Secret`: required when `rpc-secret` is configured
- `Keep file`: keep the local file downloaded by Aria2 after the transfer
