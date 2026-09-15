---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# TrustTunnel

### 结构

```yaml
type: trust-tunnel
tag: trust-tunnel-out
server: 127.0.0.1
server_port: 443
tls: {}
username: ""
password: ""
udp: false
health_check: false
quic: false
congestion_controller: ""
cwnd: 0
bbr_profile: standard
max_connections: 0
min_streams: 0
max_streams: 0
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#outbound)。

ECH 与 uTLS 引擎不兼容。

#### username

认证用户名。

#### password

认证密码。

#### udp

启用 UDP 支持。

#### health_check

启用周期性连接健康检查。

#### quic

使用 QUIC 作为传输协议（替代 TCP）。

#### congestion_controller

QUIC 拥塞控制算法，在启用 `quic` 时使用。

#### cwnd

初始拥塞窗口，在启用 `quic` 时使用。

#### bbr_profile

BBR 配置。

以下之一：`standard`、`conservative`、`aggressive`。

#### max_connections

连接池的最大连接数。

如果 `max_connections`、`min_streams`、`max_streams` 均未设置，`max_connections` 默认为 `8`。

#### min_streams

在打开新连接之前，每条连接的最小流数量。

如果 `max_connections`、`min_streams`、`max_streams` 均未设置，`min_streams` 默认为 `5`。

#### max_streams

每条连接的最大流数量。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
