---
title: "常见问题"
weight: 15
---

# 常见问题

## 上传 Alist 失败但界面显示成功

可尝试在 Alist 管理页面调小上传分片大小，或改善 Alist 部署环境的网络质量。

## Bot 提示下载成功但 Alist 未显示文件

Alist 缓存了目录结构，可参考 [Alist 缓存文档](https://alist.nn.ci/zh/guide/drivers/common.html#缓存过期) 缩短缓存过期时间或手动刷新目录。

## Docker 部署配置代理后无法连接 Telegram（初始化客户端超时）

容器默认无法直接通过 `127.0.0.1` 访问宿主机网络。可将代理地址设为宿主机真实局域网 IP，或将容器网络模式改为 `host`。