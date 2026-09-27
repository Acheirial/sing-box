# VLESS

```{.yaml linenums="1"}
type: vless
tag: vless-out

server: 127.0.0.1
server_port: 1080
uuid: bf000d23-0752-40b4-affe-68f7707a9661
flow: xtls-rprx-vision
encryption: ""
network: tcp
tls: {}
packet_encoding: ""
multiplex: {}
transport: {}

# ... 拨号字段

```

## server

**必填。**服务器地址。

## server_port

**必填。**服务器端口。

## uuid

**必填。**VLESS 用户 ID。

## flow

VLESS 子协议。

可用值：

* `xtls-rprx-vision`

## encryption

VLESS 加密，对应 Xray 的用户 `encryption`。

为空或 `none` 时不处理连接。否则使用 `mlkem768x25519plus` 语法，并在写入 VLESS 请求前加密连接：

```text
mlkem768x25519plus.<native|xorpub|random>.<1rtt|0rtt>[.<padding>][.<key>...]
```

* `native`、`xorpub` 与 `random` 选择应用于中继流量的 XOR 模式。
* `1rtt` 禁用会话票据，`0rtt` 复用会话票据。
* 其后长度小于 20 个字符的组成部分定义填充，其余为 base64url 编码的密钥（由 `xray vlessenc` 生成）。

入站的 `decryption` 必须由同一次握手生成。

## network

启用的网络协议。

`tcp` 或 `udp`。

默认所有。

## tls

TLS 配置, 参阅 [TLS](/zh/configuration/shared/tls/#出站)。

## packet_encoding

UDP 包编码，默认使用 xudp。

| 编码         | 描述            |
|------------|---------------|
| (空)        | 禁用            |
| packetaddr | 由 v2ray 5+ 支持 |
| xudp       | 由 xray 支持     |

## multiplex

参阅 [多路复用](/zh/configuration/shared/multiplex#出站)。

## transport

传输配置，参阅 [传输层](/zh/configuration/shared/transport/)。

## 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
