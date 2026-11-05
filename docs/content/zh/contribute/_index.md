---
title: "参与开发"
weight: 20
---

# 参与开发

在开始之前，请 Fork 本项目并克隆到本地，准备好 Go 开发环境。

贡献代码建议遵循以下约定，以便快速合并：

- **新功能请先提交 Issue**：便于提前讨论设计细节，避免因与整体规划不符而被拒绝。
- **代码风格**：提交前请格式化代码并保持项目整体风格一致。
- **提交规范**：遵循[语义化提交](https://www.conventionalcommits.org/zh-hans/v1.0.0/)规范。
- **测试验证**：参阅[测试文档](testing)，运行测试验证改动。

## 贡献新存储端

1. 在 `pkg/enums/storage/storages.go` 中添加新的存储端类型，并运行代码生成
2. 在 `config/storage` 目录下定义存储端配置，并添加到 `config/storage/factory.go` 中
3. 在 `storage` 目录下新建一个包编写存储端实现，然后在 `storage/storage.go` 中导入注册
4. 更新文档，添加配置说明

## 贡献新解析器

解析器可通过 Go 原生编写（推荐），也可编写为 JavaScript 插件。

若使用 Go 编写：

1. 在 `parsers` 目录下新建包编写解析器实现
2. 在 `parsers/parser.go` 的 `init` 中注册解析器

若使用 JavaScript 编写，请参考 `plugins/example_parser_basic.js` 并在该文件夹下新建 `.js` 文件实现解析逻辑。

注意：`plugins` 目录下的解析器不打包进二进制文件，需放置于配置指定的插件目录中加载。