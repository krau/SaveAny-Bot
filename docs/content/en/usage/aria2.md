---
title: "Aria2 Download"
weight: 6
---

# Aria2 Download

{{< hint warning >}}
This feature requires enabling Aria2 in the configuration file and configuring the RPC connection.
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

Add to `config.toml`:

```toml
[aria2]
enable = true
url = "http://localhost:6800/jsonrpc"
secret = "your-rpc-secret"  # If rpc-secret is configured
remove_after_transfer = true  # Remove local files after transfer
```
