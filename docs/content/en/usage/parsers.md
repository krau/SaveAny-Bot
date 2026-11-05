---
title: "Save Files Outside Telegram"
weight: 9
---

# Save Files Outside Telegram

Besides files on Telegram, the bot can also save files from other websites via JavaScript plugins or built-in parsers.

> See the [Contributing Parsers](../contribute) document for details.

Send a link that matches a parser. Built-in parsers:

- Twitter
- Kemono

## JavaScript plugin API

JavaScript plugins register one parser with:

```js
registerParser({
  metadata: {
    name: 'example',
    version: '1.0.0',
    apiVersion: '1',
    description: 'Example plugin',
    author: 'you',
  },
  canHandle(url) { return url.includes('example.com') },
  parse(url) {
    return {
      site: 'example',
      url,
      title: 'Example',
      resources: [{ url: 'https://cdn.example.com/file.zip', filename: 'file.zip' }],
    }
  },
})
```

`apiVersion` is the plugin ABI version and is independent of the plugin's own `version`. The current ABI is `"1"`; plugins with another value are rejected. The available globals are `console`, `ghttp` (`get`, `getJSON`, `head`), and `playwright`. `parse` returns an `Item` with `resources` containing `url`, `filename`, optional `mime_type`, `extension`, `size`, `hash`, `headers`, and `extra`.
