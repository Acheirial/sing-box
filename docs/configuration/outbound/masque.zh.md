---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# MASQUE

### 结构

```yaml
type: masque
tag: masque-out
server: 127.0.0.1
server_port: 443
private_key: ""
public_key: ""
ip: ""
ipv6: ""
uri: ""
mtu: 1280
udp: false
handshake_timeout: 0
skip_cert_verify: false
network: h3
congestion_controller: ""
cwnd: 0
bbr_profile: standard
remote_dns_resolve: false
dns: []
```

### 字段

#### server

服务器地址。默认使用端口 `443`。

#### server_port

服务器端口。

#### private_key

==必填==

客户端私钥。

#### public_key

==必填==

服务器公钥。

#### ip

分配给隧道的本地 IPv4 地址。

#### ipv6

分配给隧道的本地 IPv6 地址。

#### uri

MASQUE 连接 URI。默认为 `https://cloudflareaccess.com`。

#### mtu

隧道 MTU。默认为 `1280`。

#### udp

启用 UDP 支持。

#### handshake_timeout

握手超时时间（秒）。未设置时禁用。必须为非负值。

#### skip_cert_verify

跳过 TLS 证书验证。

#### network

MASQUE 网络模式。

以下之一：`h3`、`h3-l4proxy`。

`h3` 通过用户态 IP 栈转发连接（需要 `with_gvisor` 构建标签）。`h3-l4proxy` 直接转发 TCP 连接，要求使用已解析的地址。

#### congestion_controller

QUIC 拥塞控制算法。

#### cwnd

初始拥塞窗口。

#### bbr_profile

BBR 配置。

以下之一：`standard`、`conservative`、`aggressive`。

#### remote_dns_resolve

通过隧道解析 DNS 查询。

#### dns

启用 `remote_dns_resolve` 时使用的 DNS 服务器。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
