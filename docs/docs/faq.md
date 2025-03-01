# 常见问题

## 上传 alist 失败也会显示成功

这是 alist 的上传实现导致的问题, 上传到 alist 的文件实际上会被 alist 暂存在本地, 在客户端上传结束后 alist 就返回成功, 然后 alist 会在后台将文件上传到对应的存储.

目前 bot 是根据 alist 的返回判断是否成功, 无法获知 alist 的后台上传任务是否成功.

在 alist 管理页面适当调整上传分片大小, 为 alist 使用更稳定的网络环境部署, 都可以减少这种情况的发生.

## Bot 提示下载成功但是 alist 未显示

检查 alist 后台 > 任务 > 上传 中对应的上传任务的状态, 如果任务状态为成功但目录中不显示, 是由于 alist 缓存了目录结构, 参考文档可以调整缓存时间

https://alist.nn.ci/zh/guide/drivers/common.html#缓存过期

## docker部署配置了代理后仍无法连接 telegram (初始化客户端超时)

docker 不能直接访问宿主机网络, 如果你不熟悉其用法, 请将容器设为 host 模式:

