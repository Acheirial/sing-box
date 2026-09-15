### 结构

```yaml
type: vless
tag: vless-out
server: 127.0.0.1
server_port: 1080
uuid: bf000d23-0752-40b4-affe-68f7707a9661
flow: xtls-rprx-vision
testseed: []
network: tcp
tls: {}
packet_encoding: ''
encryption: ''
multiplex: {}
transport: {}
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### uuid

==必填==

VLESS 用户 ID。

#### flow

VLESS 子协议。

可用值：

* `xtls-rprx-vision`

#### testseed

!!! question "自 sing-box 1.15.0 起"

Vision 流填充参数，包含 4 个 uint32 值的列表。

仅在 `flow` 为 `xtls-rprx-vision` 时生效。

| 索引 | 含义                   | 默认值 |
|----|----------------------|-----|
| 0  | 触发长填充的内容长度阈值      | 900 |
| 1  | 长填充大小的随机范围         | 500 |
| 2  | 长填充的基数             | 900 |
| 3  | 短填充大小的随机范围         | 256 |

```yaml
testseed:
  - 900
  - 500
  - 900
  - 256
```

元素少于 4 个时回退到默认值。与 Xray 相同，不做其他校验。

#### network

启用的网络协议。

`tcp` 或 `udp`。

默认所有。

#### encryption

!!! question "自 sing-box 1.15.0 起"

VLESS 加密，即 Xray 的 `mlkem768x25519plus` 方案。

为空或 `none` 时禁用加密。

格式：`mlkem768x25519plus.<mode>.<rtt>.<key>...`

* `mode`：`native` `xorpub` `random` 之一。
* `rtt`：`1rtt` 或 `0rtt`。为 `0rtt` 时，客户端不缓存票证。
* 其后的段为 Base64（RawURL）编码的密钥（32 字节 X25519 或 1184 字节 ML-KEM-768 材料）或较短的填充片段。

服务端通过入站的 `decryption` 字段配置对应项。在入站 `users[]` 中设置 `encryption` 会报错。

#### tls

TLS 配置, 参阅 [TLS](/zh/configuration/shared/tls/#出站)。

#### packet_encoding

UDP 包编码，默认使用 xudp。

| 编码         | 描述            |
|------------|---------------|
| (空)        | 禁用            |
| packetaddr | 由 v2ray 5+ 支持 |
| xudp       | 由 xray 支持     |

#### multiplex

参阅 [多路复用](/zh/configuration/shared/multiplex#出站)。

#### transport

V2Ray 传输配置，参阅 [V2Ray 传输层](/zh/configuration/shared/v2ray-transport/)。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
