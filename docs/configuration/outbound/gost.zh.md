---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# Gost

gost relay 协议出站。

### 结构

```yaml
type: gost
tag: gost-out
server: 127.0.0.1
server_port: 1080
tls: {}
forward: false
udp: false
mux: false
username: ""
password: ""
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### tls

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#outbound)。

#### forward

启用转发模式。服务器地址仅作为中继地址使用，实际目标地址由中继服务器转发。

禁用时，每个目标都会建立到中继服务器的连接，目标地址随中继请求发送。

#### udp

启用 UDP 支持。UDP 数据包通过 UDP over TCP 协议中继。

#### mux

启用多路复用，参阅 [多路复用](/zh/configuration/shared/multiplex/)。

#### username

中继认证用户名。

#### password

中继认证密码。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
