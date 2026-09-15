---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# Mieru

### 结构

```yaml
type: mieru
tag: mieru-out
server: 127.0.0.1
server_port: 1080
port_range: ""
transport: tcp
udp: false
username: ""
password: ""
multiplexing: ""
handshake_mode: ""
traffic_pattern: ""
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

服务器端口。

与 `port_range` 冲突。`server_port` 与 `port_range` 必须设置其一。

#### port_range

服务器端口范围，格式为 `begin:end`，例如 `10000:20000`。

与 `server_port` 冲突。`server_port` 与 `port_range` 必须设置其一。

#### transport

传输协议。

以下之一：`tcp`、`udp`。

#### udp

启用 UDP 支持。

#### username

用户名。

#### password

密码。

#### multiplexing

多路复用级别。

以下之一：`MULTIPLEXING_DEFAULT`、`MULTIPLEXING_OFF`、`MULTIPLEXING_LOW`、`MULTIPLEXING_MIDDLE`、`MULTIPLEXING_HIGH`。

#### handshake_mode

握手模式。

以下之一：`HANDSHAKE_DEFAULT`、`HANDSHAKE_STANDARD`、`HANDSHAKE_NO_WAIT`。

`HANDSHAKE_STANDARD` 也称为 1-RTT：客户端等待代理服务器建立到目标的连接后再发送数据。`HANDSHAKE_NO_WAIT` 也称为 0-RTT：客户端在连接代理服务器的同时发送数据。

#### traffic_pattern

流量模式，以 base64 字符串编码。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
