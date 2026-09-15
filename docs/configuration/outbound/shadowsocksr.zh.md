---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# ShadowsocksR

### 结构

```yaml
type: shadowsocksr
tag: ssr-out
server: 127.0.0.1
server_port: 1080
method: aes-256-cfb
password: hello
obfs: plain
obfs_param: ""
protocol: origin
protocol_param: ""
network: tcp
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### method

==必填==

加密方法。

以下之一：

* `rc4-md5`
* `aes-128-ctr`
* `aes-192-ctr`
* `aes-256-ctr`
* `aes-128-cfb`
* `aes-192-cfb`
* `aes-256-cfb`
* `chacha20`
* `chacha20-ietf`
* `xchacha20`
* `none`

`none` 是 dummy 加密的别名。

#### password

==必填==

ShadowsocksR 密码。

#### obfs

混淆方法。

以下之一：`plain`、`http_simple`、`http_post`、`random_head`、`tls1.2_ticket_auth`、`tls1.2_ticket_fastauth`。

`plain` 表示不混淆。

#### obfs_param

混淆方法的参数。

#### protocol

ShadowsocksR 协议。

以下之一：`origin`、`auth_sha1_v4`、`auth_aes128_md5`、`auth_aes128_sha1`、`auth_chain_a`、`auth_chain_b`。

`origin` 表示不使用协议插件。

#### protocol_param

协议的参数。

#### network

启用的网络。

`tcp` `udp` 之一。

默认同时启用。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
