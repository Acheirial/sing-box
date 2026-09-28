# TLS

## 入站

```{.yaml linenums="1"}
enabled: true
server_name: ""
alpn: []
min_version: ""
max_version: ""
cipher_suites: []
curve_preferences: []
certificate: []
certificate_path: ""
client_authentication: ""
client_certificate: []
client_certificate_path: []
client_certificate_sha256: []
client_certificate_public_key_sha256: []
key: []
key_path: ""
kernel_tx: false
kernel_rx: false
handshake_timeout: ""
certificate_provider: ""

ech:
  enabled: false
  key: []
  key_path: ""
reality:
  enabled: false
  handshake:
    server: google.com
    server_port: 443

    # ... 拨号字段
  private_key: UuMBgl7MXTPx9inmQp2UC7Jcnwc6XYbwDNebonM-FCc
  short_id:
    - 0123456789abcdef
  max_time_difference: 1m
  mldsa65_seed: ""
```

## 出站

```{.yaml linenums="1"}
enabled: true
engine: ""
disable_sni: false
server_name: ""
insecure: false
alpn: []
min_version: ""
max_version: ""
cipher_suites: []
curve_preferences: []
certificate: ""
certificate_path: ""
certificate_sha256: []
certificate_public_key_sha256: []
client_certificate: []
client_certificate_path: ""
client_key: []
client_key_path: ""
fragment: false
fragment_fallback_delay: ""
record_fragment: false
spoof: ""
spoof_method: ""
kernel_tx: false
kernel_rx: false
handshake_timeout: ""
ech:
  enabled: false
  config: []
  config_path: ""
  query_server_name: ""
utls:
  enabled: false
  fingerprint: ""
reality:
  enabled: false
  public_key: jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0
  short_id: 0123456789abcdef
  mldsa65_verify: ""
```

TLS 版本值：

* `1.0`
* `1.1`
* `1.2`
* `1.3`

密码套件值：

* `TLS_RSA_WITH_AES_128_CBC_SHA`
* `TLS_RSA_WITH_AES_256_CBC_SHA`
* `TLS_RSA_WITH_AES_128_GCM_SHA256`
* `TLS_RSA_WITH_AES_256_GCM_SHA384`
* `TLS_AES_128_GCM_SHA256`
* `TLS_AES_256_GCM_SHA384`
* `TLS_CHACHA20_POLY1305_SHA256`
* `TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA`
* `TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA`
* `TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA`
* `TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA`
* `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`
* `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`
* `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
* `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`
* `TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256`
* `TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256`

!!! note ""

    当内容只有一项时，可以直接使用单个值，无需数组

## enabled

启用 TLS

## engine

**仅客户端。**要使用的 TLS 引擎。

可用值：

* `go`（默认）
* `apple`
* `windows`

支持的字段：

* `server_name`
* `insecure`
* `alpn`
* `min_version`
* `max_version`
* `certificate` / `certificate_path`
* `certificate_sha256`
* `certificate_public_key_sha256`
* `handshake_timeout`

不支持的字段：

* `disable_sni`
* `cipher_suites`
* `curve_preferences`
* `client_certificate` / `client_certificate_path` / `client_key` / `client_key_path`
* `fragment` / `record_fragment`
* `kernel_tx` / `kernel_rx`
* `ech`
* `utls`
* `reality`

!!! note ""

    `windows` 通过 SSPI 使用 Schannel，仅在 Windows build 17763 及以上可用，包括 Windows 10 版本 1809、Windows Server 2019 及后续版本。

!!! note ""

    TLS 1.3 仅在 Windows 11 或 Windows Server 2022 及后续版本上协商。

默认版本范围为 TLS 1.2 到 TLS 1.3，与 `go` 引擎一致。

## disable_sni

**仅客户端。**不要在 ClientHello 中发送服务器名称.

## server_name

用于验证返回证书上的主机名，除非设置不安全。

它还包含在 ClientHello 中以支持虚拟主机，除非它是 IP 地址。

## insecure

**仅客户端。**接受任何服务器证书。

## alpn

支持的应用层协议协商列表，按优先顺序排列。

如果两个对等点都支持 ALPN，则选择的协议将是此列表中的一个，如果没有相互支持的协议则连接将失败。

参阅 [Application-Layer Protocol Negotiation](https://en.wikipedia.org/wiki/Application-Layer_Protocol_Negotiation)。

## min_version

可接受的最低 TLS 版本。

默认使用 TLS 1.2。

## max_version

可接受的最大 TLS 版本。

默认情况下，当前最高版本为 TLS 1.3。

## cipher_suites

启用的 TLS 1.0–1.2 密码套件列表。列表的顺序被忽略。请注意，TLS 1.3 的密码套件是不可配置的。

如果为空，则使用安全的默认列表。默认密码套件可能会随着时间的推移而改变。

## curve_preferences

支持的密钥交换机制集合。列表的顺序被忽略，密钥交换机制通过 Golang 的内部偏好顺序从此列表中选择。

可用值，同时也是默认列表：

* `P256`
* `P384`
* `P521`
* `X25519`
* `X25519MLKEM768`

## certificate

服务器证书链行数组，PEM 格式。

## certificate_path

!!! note ""

    文件更改时将自动重新加载。

服务器证书链路径，PEM 格式。

## certificate_sha256

**仅客户端。**服务器证书的 SHA-256 哈希列表，base64 格式。

哈希基于整个 DER 编码的证书计算。如果只需要固定公钥，请使用 `certificate_public_key_sha256`。

要生成证书的 SHA-256 哈希，请使用以下命令：

```bash
# 对于证书文件
openssl x509 -in certificate.pem -outform der | openssl dgst -sha256 -binary | openssl enc -base64

# 对于远程服务器的证书
echo | openssl s_client -servername example.com -connect example.com:443 2>/dev/null | openssl x509 -outform der | openssl dgst -sha256 -binary | openssl enc -base64
```

## certificate_public_key_sha256

**仅客户端。**服务器证书公钥的 SHA-256 哈希列表，base64 格式。

要生成证书公钥的 SHA-256 哈希，请使用以下命令：

```bash

# 对于证书文件

openssl x509 -in certificate.pem -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64

# 对于远程服务器的证书

echo | openssl s_client -servername example.com -connect example.com:443 2>/dev/null | openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64
```

## client_certificate

**仅客户端。**客户端证书链行数组，PEM 格式。

## client_certificate_path

**仅客户端。**客户端证书链路径，PEM 格式。

## client_key

**仅客户端。**客户端私钥行数组，PEM 格式。

## client_key_path

**仅客户端。**客户端私钥路径，PEM 格式。

## key

**仅服务器。**

!!! note ""

    文件更改时将自动重新加载。

服务器 PEM 私钥行数组。

## key_path

**仅服务器。**

!!! note ""

    文件更改时将自动重新加载。

服务器私钥路径，PEM 格式。

## client_authentication

**仅服务器。**要使用的客户端身份验证类型。

可用值：

* `no`（默认）
* `request`
* `require-any`
* `verify-if-given`
* `require-and-verify`

如果此选项设置为 `verify-if-given` 或 `require-and-verify`，
则需要 `client_certificate`、`client_certificate_path`、`client_certificate_sha256` 或 `client_certificate_public_key_sha256` 中的一个。

## client_certificate

**仅服务器。**客户端证书链行数组，PEM 格式。

## client_certificate_path

**仅服务器。**

!!! note ""

    文件更改时将自动重新加载。

客户端证书链路径列表，PEM 格式。

## client_certificate_sha256

**仅服务器。**客户端证书的 SHA-256 哈希列表，base64 格式。

哈希基于整个 DER 编码的证书计算，参阅 [certificate_sha256](#certificate_sha256)。

## client_certificate_public_key_sha256

**仅服务器。**客户端证书公钥的 SHA-256 哈希列表，base64 格式。

要生成证书公钥的 SHA-256 哈希，请使用以下命令：

```bash

# 对于证书文件

openssl x509 -in certificate.pem -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64

# 对于远程服务器的证书

echo | openssl s_client -servername example.com -connect example.com:443 2>/dev/null | openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64
```

## kernel_tx

!!! quote ""

    仅支持 Linux 5.1+，如果可能，使用较新的内核。

!!! quote ""

    仅支持 TLS 1.3。

!!! warning ""

    kTLS TX 仅当 `splice(2)` 可用时（两端经过握手后必须为没有附加协议的 TCP 或 TLS）才能提高性能；否则肯定会降低性能。

启用内核 TLS 发送支持。

## kernel_rx

!!! quote ""

    仅支持 Linux 5.1+，如果可能，使用较新的内核。

!!! quote ""

    仅支持 TLS 1.3。

!!! failure ""

    即使使用 `splice(2)`，kTLS RX 也肯定会降低性能，因此不建议启用。

启用内核 TLS 接收支持。

## handshake_timeout

TLS 握手超时，采用 golang 的 Duration 格式。

默认使用 `15s`。

## certificate_provider

**仅服务器。**字符串或对象。

为字符串时，共享[证书提供者](/zh/configuration/shared/certificate-provider/)的标签。

为对象时，内联的证书提供者。可用类型和字段参阅[证书提供者](/zh/configuration/shared/certificate-provider/)。

## 自定义 TLS 支持

!!! info "QUIC 支持"

    只有 ECH 在 QUIC 中被支持.

### utls

**仅客户端。**

!!! failure "不推荐"

    uTLS 已被研究人员多次发现其指纹可被识别的漏洞。

    uTLS 是一个试图通过复制 ClientHello 结构来模仿浏览器 TLS 指纹的 Go 库。
    然而，浏览器使用完全不同的 TLS 实现（Chrome 使用 BoringSSL，Firefox 使用 NSS），
    其实现行为无法通过简单复制握手格式来复现，其行为细节必然存在差异，使得检测成为可能。
    此外，此库缺乏积极维护，且代码质量较差，不建议用于反审查场景。

    如需 TLS 指纹抵抗，请改用 [NaiveProxy](/zh/configuration/inbound/naive/)。

uTLS 是 "crypto/tls" 的一个分支，它提供了 ClientHello 指纹识别阻力。

可用的指纹值：

* chrome
* firefox
* edge
* safari
* 360
* qq
* ios
* android
* random
* randomized

默认使用 chrome 指纹。

## ECH 字段

ECH (Encrypted Client Hello) 是一个 TLS 扩展，它允许客户端加密其 ClientHello 的第一部分信息。

ECH 密钥和配置可以通过 `sing-box generate ech-keypair` 生成。

### key

**仅服务器。**ECH 密钥行数组，PEM 格式。

### key_path

**仅服务器。**

!!! note ""

    文件更改时将自动重新加载。

ECH 密钥路径，PEM 格式。

### config

**仅客户端。**ECH 配置行数组，PEM 格式。

如果为空，将尝试从 DNS 加载。

### config_path

**仅客户端。**ECH 配置路径，PEM 格式。

如果为空，将尝试从 DNS 加载。

### query_server_name

**仅客户端。**覆盖用于 ECH HTTPS 记录查询的域名。

如果为空，使用 `server_name` 查询。

### fragment

**仅客户端。**通过分段 TLS 握手数据包来绕过防火墙。

此功能旨在规避基于**明文数据包匹配**的简单防火墙，不应该用于规避真正的审查。

由于性能不佳，请首先尝试 `record_fragment`，且仅应用于已知被阻止的服务器名称。

在 Linux、Apple 平台和（需要管理员权限的）Windows 系统上，
可以自动检测等待时间。否则，将回退到
等待 `fragment_fallback_delay` 指定的固定时间。

此外，如果实际等待时间少于 20ms，也会回退到等待固定时间，
因为目标被认为是本地的或在透明代理后面。

### fragment_fallback_delay

**仅客户端。**当 TLS 分段无法自动确定等待时间时使用的回退值。

默认使用 `500ms`。

### record_fragment

**仅客户端。**将 TLS 握手分段为多个 TLS 记录以绕过防火墙。

### spoof

**仅客户端。**

!!! quote ""

    仅支持 Linux、macOS 和 Windows，需要提升的权限。

在真实 ClientHello 之前注入一个伪造的、携带白名单 SNI 的 TLS ClientHello，
以欺骗基于 SNI 过滤的中间盒放行连接。

Linux 上需要 `CAP_NET_RAW` 和 `CAP_NET_ADMIN`，macOS 上需要 root，Windows 上需要 Administrator。
不支持 Windows ARM64。

### spoof_method

**仅客户端。**控制伪造报文被真实服务器拒绝的方式。

| 取值                     | 行为                                                              |
|--------------------------|-------------------------------------------------------------------|
| `wrong-sequence`（默认） | 伪造报文的 TCP 序列号位于服务器接收窗口之前。                     |
| `wrong-checksum`         | 伪造报文的 TCP 校验和被故意设为无效。                             |
| `wrong-ack`              | 伪造报文的 TCP 确认号位于服务器发送窗口之前。                     |
| `wrong-md5`              | 伪造报文携带 TCP-MD5 签名选项。                                   |
| `wrong-timestamp`        | 伪造报文携带回退的 TCP 时间戳。仅支持 Linux/Windows，不支持 macOS。 |

## Reality 字段

### handshake

**仅服务器。**

**必填。**握手服务器地址和 [拨号参数](/zh/configuration/shared/dial/)。

### private_key

**仅服务器。**

**必填。**私钥，由 `sing-box generate reality-keypair` 生成。

### server_names

**仅服务器。**除 `server_name` 之外，服务器额外接受的 TLS 服务器名称（SNI）。默认空。

### min_client_ver

**仅服务器。**接受的最小 REALITY 客户端版本，格式为 `major.minor.patch`。为空表示不限制。

### max_client_ver

**仅服务器。**接受的最大 REALITY 客户端版本，格式为 `major.minor.patch`。为空表示不限制。

### limit_fallback_upload

**仅服务器。**应用于回退连接上传流量的速率限制：

* `after_bytes`：开始限速前允许传输的字节数。
* `bytes_per_sec`：持续速率。
* `burst_bytes_per_sec`：突发速率。

### limit_fallback_download

**仅服务器。**应用于回退连接下载流量的速率限制，字段与 `limit_fallback_upload` 相同。

### xver

**仅服务器。**为回退连接请求的 PROXY 协议版本（`0`、`1` 或 `2`）。配置后，在回退连接上发送 PROXY 协议（v1 或 v2）。

### mldsa65_seed

**仅服务器。**用于 ML-DSA-65 服务器证书签名的 32 字节 Base64 URL 编码种子。

### show

**仅服务器。**以 info 级别而非 trace 级别记录 REALITY 握手详情。默认关闭。

### master_key_log

**仅服务器。**以 NSS key log 格式写入 TLS 主密钥的文件，`none` 表示禁用。

### public_key

**仅客户端。**

**必填。**公钥，由 `sing-box generate reality-keypair` 生成。

### short_id

**必填。**一个零到八位的十六进制字符串。

### max_time_difference

**仅服务器。**服务器和客户端之间的最大时间差。

如果为空则禁用检查。

### mldsa65_verify

**仅客户端。**Base64 格式的 1952 字节 ML-DSA-65 公钥，用于验证证书第一个扩展中的额外后量子签名。

### spider_x

**仅客户端。**验证失败后爬取回退服务器时使用的初始路径。`p`、`c`、`t`、`i` 与 `r` 查询参数用于配置 SpiderY，含义见 Xray-core REALITY 文档。
