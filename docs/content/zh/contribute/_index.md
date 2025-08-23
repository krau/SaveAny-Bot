---
title: "参与开发"
weight: 20
---

# 参与开发

在开始之前, 请 Fork 本项目, 并克隆到本地, 并确保 Go 版本 >= 1.23. 

以下是一些贡献代码的指南或建议, 你不必完全遵守, 但将有助于快速 review 并合并你的提交:

- **新功能请先提交 Issue**, 以便讨论设计和实现细节, 并避免因与项目设计不符而被拒绝.
- **使用现代开发工具**, 确保提交前格式化代码, 并保持风格一致.
- **使用[语义化提交](https://www.conventionalcommits.org/zh-hans/v1.0.0/)**, 避免提交消息模糊或过于简单.

## 贡献新存储端

1. 在 `pkg/enums/storage/storages.go` 中添加新的存储端类型, 并运行代码生成
2. 在 `config/storage` 目录下定义存储端配置, 并添加到 `config/storage/factory.go` 中
3. 在 `storage` 目录下新建一个包, 编写存储端实现, 然后在 `storage/storage.go` 中导入并添加它
4. 更新文档, 添加配置说明

## 贡献新解析器

你可以选择使用 Go 编写原生的解析器实现(推荐), 或是使用 JavaScript 以插件的方式实现.

如果使用 Go 编写, 请:

1. 在 `parsers` 目录下新建一个包, 编写解析器实现
2. 在 `parsers/parser.go` 的 `init` 中注册解析器

如果使用 JavaScript 编写, 请参考 `plugins/example_parser_basic.js` 的实现, 并在该文件夹下新建一个 js 文件, 实现你的解析逻辑.

需要注意, `plugins` 目录下解析器默认不会被编译到二进制文件中, 用户需要手动下载它们并放到本地指定目录下以启用它们.