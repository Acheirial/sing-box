---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# ShadowQUIC

### 结构

```yaml
type: shadowquic
tag: shadowquic-out
server: 127.0.0.1
server_port: 443
tls: {}
username: ""
password: ""
alpn:
  - h3
quic_versions:
  - v1
udp_over_stream: false
zero_rtt: false
keep_alive_interval: 0s
congestion_controller: bbr_meta_v2
up: ""
down: ""
cwnd: 32
bbr_profile: standard
receive_window_conn: 0
receive_window: 0
disable_mtu_discovery: false
max_datagram_frame_size: 1400
max_open_streams: 1024
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

服务器端口。默认为 `443`。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#outbound)。

要求使用标准 TLS 引擎。

#### username

JLS 认证用户名。

!!! note

    此构建不支持 JLS 认证（用户名/密码）。

#### password

JLS 认证密码。

#### alpn

TLS ALPN 值。默认为 `h3`。

#### quic_versions

QUIC 版本。

以下之一：`v1`、`v2`。

默认使用 `v1`。

#### udp_over_stream

启用 UDP over stream 中继模式，提供基于 QUIC 流的 UDP 中继。

#### zero_rtt

启用 QUIC 0-RTT。DialEarly 只有在 TLS 缓存了此前与该服务器连接的会话票据后才能发送 0-RTT 数据。

#### keep_alive_interval

QUIC keep alive 间隔。未设置时禁用。

#### congestion_controller

QUIC 拥塞控制算法。

以下之一：`cubic`、`new_reno`、`bbr_meta_v1`、`bbr_meta_v2`、`bbr`。

#### up

上传带宽，供 BBR 拥塞控制器使用，例如 `100 mbps`。

#### down

下载带宽，供 BBR 拥塞控制器使用，例如 `100 mbps`。

#### cwnd

初始拥塞窗口。默认为 `32`。

#### bbr_profile

BBR 配置。

以下之一：`standard`、`conservative`、`aggressive`。

#### receive_window_conn

QUIC 流接收窗口。

#### receive_window

QUIC 连接接收窗口。

#### disable_mtu_discovery

禁用 QUIC 路径 MTU 发现。

#### max_datagram_frame_size

最大 QUIC 数据报帧大小。默认为 `1400`。

#### max_open_streams

最大 QUIC 流数量。默认为 `1024`。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
