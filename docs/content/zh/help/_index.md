---
title: "常见问题"
weight: 15
---

# 常见问题

## 上传 alist 失败也会显示成功

在 alist 管理页面适当调整上传分片大小, 为 alist 使用更稳定的网络环境部署, 都可以减少这种情况的发生.

## Bot 提示下载成功但是 alist 未显示

alist 缓存了目录结构, 参考 <a href="https://alist.nn.ci/zh/guide/drivers/common.html#缓存过期" target="_blank">文档</a> 可以调整缓存时间

## docker部署配置了代理后仍无法连接 telegram (初始化客户端超时)

docker 不能直接访问宿主机网络, 如果你不熟悉其用法, 请将容器设为 host 模式.