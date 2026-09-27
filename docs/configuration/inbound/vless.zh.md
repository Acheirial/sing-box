# VLESS

```{.yaml linenums="1"}
type: vless
tag: vless-in

decryption: ""

# ... 监听字段

users:
  - name: sekai
    uuid: bf000d23-0752-40b4-affe-68f7707a9661
    flow: ""
tls: {}
multiplex: {}
transport: {}
```

## 监听字段

参阅 [监听字段](/zh/configuration/shared/listen/)。

## users

**必填。**VLESS 用户。

## users.uuid

**必填。**VLESS 用户 ID。

## users.flow

VLESS 子协议。

可用值：

* `xtls-rprx-vision`

## decryption

VLESS 加密，对应 Xray 的 `decryption`。

为空或 `none` 时不处理连接。否则使用 `mlkem768x25519plus` 语法，并在读取 VLESS 请求前解密连接：

```text
mlkem768x25519plus.<native|xorpub|random>.<from>[s]|[-<to>[s]][.<padding>][.<key>...]
```

* `native`、`xorpub` 与 `random` 选择应用于中继流量的 XOR 模式。
* 第二个组成部分为票据有效期（秒），可以是单个值或范围，如 `600s` 或 `600-1200s`。
* 其后长度小于 20 个字符的组成部分定义填充，其余为 base64url 编码的密钥（由 `xray vlessenc` 生成）。

## tls

TLS 配置, 参阅 [TLS](/zh/configuration/shared/tls/#入站)。

## multiplex

参阅 [多路复用](/zh/configuration/shared/multiplex#入站)。

## transport

传输配置，参阅 [传输层](/zh/configuration/shared/transport/)。
