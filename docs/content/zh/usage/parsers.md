---
title: "转存 Telegram 之外的文件"
weight: 9
---

# 转存 Telegram 之外的文件

除了 Telegram 上的文件, Bot 还可通过 JavaScript 插件或内置解析器来支持转存其他网站的文件.

> 查看[贡献解析器](../contribute)文档了解详情

向 Bot 发送符合解析器要求的链接。内置解析器：

- Twitter
- Kemono

## JavaScript 插件 API

JavaScript 插件通过以下结构注册解析器：

```js
registerParser({
  metadata: {
    name: 'example',
    version: '1.0.0',
    apiVersion: '1',
    description: '示例插件',
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

`apiVersion` 是插件 ABI 版本，与插件自身的 `version` 独立。当前 ABI 为 `"1"`，其他值会被拒绝。可用全局对象为 `console`、`ghttp`（`get`、`getJSON`、`head`）和 `playwright`。`parse` 返回 `Item`，其中 `resources` 支持 `url`、`filename`，以及可选的 `mime_type`、`extension`、`size`、`hash`、`headers`、`extra`。
