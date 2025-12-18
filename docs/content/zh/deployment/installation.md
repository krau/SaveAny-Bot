---
title: "安装与更新"
---

# 安装与更新

## 从预编译文件部署(推荐)

在 [Release](https://github.com/krau/SaveAny-Bot/releases) 页面下载对应平台的二进制文件.

在解压后目录新建 `config.toml` 文件, 参考 [配置说明](../configuration) 编辑配置文件

运行:

```bash
chmod +x saveany-bot
./saveany-bot
```

### 进程守护

{{< tabs "daemon" >}}
{{< tab "systemd (常规 Linux)" >}}

创建文件 <code>/etc/systemd/system/saveany-bot.service</code> 并写入以下内容:

{{< codeblock >}}
[Unit]
Description=SaveAnyBot
After=systemd-user-sessions.service

[Service]
Type=simple
WorkingDirectory=/yourpath/
ExecStart=/yourpath/saveany-bot
Restart=always

[Install]
WantedBy=multi-user.target
{{< /codeblock >}}

设为开机启动并启动服务:

{{< codeblock >}}
systemctl enable --now saveany-bot
{{< /codeblock >}}

{{< /tab >}}

{{< tab "procd (OpenWrt)" >}}

<h4>添加开机自启动服务</h4>

创建文件 <code>/etc/init.d/saveanybot</code> ，参考 <a href="https://github.com/krau/SaveAny-Bot/blob/main/docs/confs/wrt_init" target="_blank">wrt_init</a> 并自行修改:

{{< codeblock >}}
#!/bin/sh /etc/rc.common

#This is the OpenWRT init.d script for SaveAnyBot

START=99 
STOP=10
description="SaveAnyBot"

WORKING_DIR="/mnt/mmc1-1/SaveAnyBot"
EXEC_PATH="$WORKING_DIR/saveany-bot"
start() {
    echo "Starting SaveAnyBot..."
    cd $WORKING_DIR
    $EXEC_PATH &
}
stop() {
    echo "Stopping SaveAnyBot..."
    killall saveany-bot
}
reload() {
    stop
    start
}

{{< /codeblock >}}

赋予权限:

{{< codeblock >}}
chmod +x /etc/init.d/saveanybot
{{< /codeblock >}}

然后将文件复制到 <code>/etc/rc.d</code> 并重命名为 <code>S99saveanybot</code>, 同样赋予权限:

{{< codeblock >}}
chmod +x /etc/rc.d/S99saveanybot
{{< /codeblock >}}

<h4>添加快捷指令</h4>

创建文件 <code>/usr/bin/sabot</code> ，参考 <a href="https://github.com/krau/SaveAny-Bot/blob/main/docs/confs/wrt_bin" target="_blank">wrt_bin</a>  并自行修改，注意此处文件编码仅支持 ANSI 936 .

随后赋予权限:

{{< codeblock >}}
chmod +x /usr/bin/sabot
{{< /codeblock >}}

使用: <code>sudo sabot start|stop|restart|status|enable|disable</code>

{{< /tab >}}
{{< /tabs >}}


## 使用 Docker 部署

### Docker Compose

下载 [docker-compose.yml](https://github.com/krau/SaveAny-Bot/blob/main/docker-compose.yml) 文件, 在同目录下新建 `config.toml` 文件, 参考 [config.example.toml](https://github.com/krau/SaveAny-Bot/blob/main/config.example.toml) 编辑配置文件.

启动:

```bash
docker compose up -d
```

### Docker

```shell
docker run -d --name saveany-bot \
    -v /path/to/config.toml:/app/config.toml \
    -v /path/to/downloads:/app/downloads \
    ghcr.io/krau/saveany-bot:latest
```

{{< hint info >}}
关于 docker 镜像的变体版本
<br />
<ul>
<li>默认版本: 包含所有功能和依赖, 体积较大. 如果没有特殊需要, 请使用此版本</li>
<li>micro: 精简版本, 去除部分可选依赖, 体积较小</li>
<li>pico: 极简版本, 仅包含核心功能, 体积最小</li>
</ul>
你可以根据需要, 通过指定不同的标签来拉取合适的版本, 例如: <code>ghcr.io/krau/saveany-bot:micro</code>
<br />
关于变体版本的更详细的区别, 请参考项目根目录下的 Dockerfile 文件.
{{< /hint >}}

## 更新

若使用预编译二进制文件部署, 使用以下 CLI 命令更新:

```bash
./saveany-bot up
```

如果是 Docker 部署, 使用以下命令更新:

docker:

```bash
docker pull ghcr.io/krau/saveany-bot:latest
docker restart saveany-bot
```

docker compose:

```bash
docker compose pull
docker compose restart
```