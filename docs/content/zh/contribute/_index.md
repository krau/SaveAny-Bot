---
title: "参与开发"
weight: 20
---

# 参与开发

## 贡献新存储端

1. Fork 本项目, 克隆到本地
2. 在 `pkg/enums/storage/storages.go` 中添加新的存储端类型, 并运行代码生成
3. 在 `config/storage` 目录下定义存储端配置, 并添加到 `config/storage/factory.go` 中
4. 在 `storage` 目录下新建一个包, 编写存储端实现, 然后在 `storage/storage.go` 中导入并添加它
5. 更新文档, 添加配置说明