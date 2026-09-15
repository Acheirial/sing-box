V2Ray Transport 是 v2ray 发明的一组私有协议，并污染了其他协议的名称，如 clash 中的 `trojan-grpc`。

### 结构

```yaml
type: ''
```

可用的传输协议：

* HTTP
* WebSocket
* QUIC
* gRPC
* HTTPUpgrade
* XHTTP

!!! warning "与 v2ray-core 的区别"

    * 没有 TCP 传输层, 纯 HTTP 已合并到 HTTP 传输层。
    * 没有 mKCP 传输层。
    * 没有 DomainSocket 传输层。

!!! note ""

    当内容只有一项时，可以忽略 JSON 数组 [] 标签。

### HTTP

```yaml
type: http
host: []
path: ''
method: ''
headers: {}
idle_timeout: 15s
ping_timeout: 15s
```

!!! warning "与 v2ray-core 的区别"

    不强制执行 TLS。如果未配置 TLS，将使用纯 HTTP 1.1。

#### host

主机域名列表。

如果设置，客户端将随机选择，服务器将验证。

#### path

!!! warning

    V2Ray 文档称服务端和客户端的路径必须一致，但实际代码允许客户端向路径添加任何后缀。
    sing-box 使用与 V2Ray 相同的行为，但请注意，该行为在 `WebSocket` 和 `HTTPUpgrade` 传输层中不存在。

HTTP 请求路径

服务器将验证。

#### method

HTTP 请求方法

如果设置，服务器将验证。

#### headers

HTTP 请求的额外标头

如果设置，服务器将写入响应。

#### idle_timeout

在 HTTP2 服务器中：

指定闲置客户端应在多长时间内使用 GOAWAY 帧关闭。PING 帧不被视为活动。

在 HTTP2 客户端中：

如果连接上没有收到任何帧，指定一段时间后将使用 PING 帧执行健康检查。需要注意的是，PING 响应被视为已接收的帧，因此如果连接上没有其他流量，则健康检查将在每个间隔执行一次。如果值为零，则不会执行健康检查。

默认使用零。

#### ping_timeout

在 HTTP2 客户端中：

指定发送 PING 帧后，在指定的超时时间内必须接收到响应。如果在指定的超时时间内没有收到 PING 帧的响应，则连接将关闭。默认超时持续时间为 15 秒。

### WebSocket

```yaml
type: ws
path: ''
headers: {}
max_early_data: 0
early_data_header_name: ''
```

#### path

HTTP 请求路径

服务器将验证。

#### headers

HTTP 请求的额外标头

如果设置，服务器将写入响应。

#### max_early_data

请求中允许的最大有效负载大小。默认启用。

#### early_data_header_name

默认情况下，早期数据在路径而不是标头中发送。

要与 Xray-core 兼容，请将其设置为 `Sec-WebSocket-Protocol`。

它需要与服务器保持一致。

### QUIC

```yaml
type: quic
```

!!! warning "与 v2ray-core 的区别"

    没有额外的加密支持：
    它基本上是重复加密。 并且 Xray-core 在这里与 v2ray-core 不兼容。

### gRPC

!!! note ""

    默认安装不包含标准 gRPC (兼容性好，但性能较差), 参阅 [安装](/zh/installation/build-from-source/#构建标记)。

```yaml
type: grpc
service_name: TunService
idle_timeout: 15s
ping_timeout: 15s
permit_without_stream: false
```

#### service_name

gRPC 服务名称。

#### idle_timeout

在标准 gRPC 服务器/客户端：

如果传输在此时间段后没有看到任何活动，它会向客户端发送 ping 请求以检查连接是否仍然活动。

在默认 gRPC 服务器/客户端：

它的行为与 HTTP 传输层中的相应设置相同。

#### ping_timeout

在标准 gRPC 服务器/客户端：

经过一段时间之后，客户端将执行 keepalive 检查并等待活动。如果没有检测到任何活动，则会关闭连接。

在默认 gRPC 服务器/客户端：

它的行为与 HTTP 传输层中的相应设置相同。

#### permit_without_stream

在标准 gRPC 客户端：

如果启用，客户端传输即使没有活动连接也会发送 keepalive ping。如果禁用，则在没有活动连接时，将忽略 `idle_timeout` 和 `ping_timeout`，并且不会发送 keepalive ping。

默认禁用。

### HTTPUpgrade

```yaml
type: httpupgrade
host: ''
path: ''
headers: {}
```

#### host

主机域名。

服务器将验证。

#### path

HTTP 请求路径

服务器将验证。

#### headers

HTTP 请求的额外标头。

如果设置，服务器将写入响应。

### XHTTP

!!! question "自 sing-box 1.15.0 起"

```yaml
type: xhttp
host: ''
path: ''
mode: ''
headers: {}
alpn: []
no_grpc_header: false
x_padding_bytes: ''
x_padding_obfs_mode: false
x_padding_key: ''
x_padding_header: ''
x_padding_placement: ''
x_padding_method: ''
uplink_http_method: ''
session_placement: ''
session_key: ''
session_table: ''
session_length: ''
seq_placement: ''
seq_key: ''
uplink_data_placement: ''
uplink_data_key: ''
uplink_chunk_size: ''
sc_max_each_post_bytes: ''
sc_min_posts_interval_ms: ''
reuse_settings:
  max_concurrency: ''
  max_connections: ''
  c_max_reuse_times: ''
  h_max_request_times: ''
  h_max_reusable_secs: ''
  h_keep_alive_period: 0
no_sse_header: false
sc_stream_up_server_secs: ''
sc_max_buffered_posts: ''
download_settings:
  path: ''
  host: ''
  headers: {}
  reuse_settings: {}
  server: ''
  server_port: 0
  tls: {}
  alpn: []
extra: {}
```

#### host

主机域名。

#### path

HTTP 请求路径。

#### mode

XHTTP 传输模式。

`auto` `packet-up` `stream-up` `stream-one` 之一。

默认使用 `auto`。为 `auto` 时，配合 REALITY 使用 `stream-one`，配合 REALITY 和 `download_settings` 使用
`stream-up`，否则使用 `packet-up`。

#### extra

一个在解析后叠加到传输配置上的对象。`extra` 中设置的字段会覆盖对应的顶层字段，为较新的 Xray 选项提供前向兼容。

`host`、`path` 和 `mode` 不受覆盖影响。未知的键将被忽略。

#### headers

HTTP 请求的额外标头。

#### alpn

HTTP 传输的 ALPN 列表。

未设置时默认使用 TLS 配置中的 ALPN。

#### no_grpc_header

默认情况下，带有请求体的 stream-up/stream-one 请求会附加 `Content-Type: application/grpc` 标头。

设置为 `true` 可省略该标头。

#### x_padding_bytes

填充长度，格式为 `100-1000` 的范围字符串或单个值。

默认使用 `100-1000`。

#### x_padding_obfs_mode

启用混淆填充模式，可自定义填充的放置位置、键、标头和方法。

禁用时使用旧版格式：客户端请求在 `Referer` 标头的 URL 查询参数中携带名为 `x_padding` 的填充，服务器通过
`X-Padding` 响应标头返回填充。

#### x_padding_key

填充位于 cookie 或查询参数中时使用的参数名。

旧版填充格式使用 `x_padding`。

#### x_padding_header

填充位于标头中时使用的标头名。

#### x_padding_placement

填充的放置位置。

`header` `queryInHeader` `cookie` `query` 之一。

#### x_padding_method

填充生成方式。

`repeat-x` `tokenish` 之一。

`repeat-x` 使用重复的 `X` 字符填充；`tokenish` 生成看似 token 的随机 base62 字符串。为空时使用 `repeat-x`。

#### uplink_http_method

上行请求使用的 HTTP 方法。

默认使用 `POST`。

#### session_placement

会话 ID 的放置位置。

`path` `query` `header` `cookie` 之一。

默认使用 `path`。

#### session_key

会话 ID 位于查询参数、标头或 cookie 中时使用的参数名。

默认值取决于放置位置：标头为 `X-Session`，cookie 和查询为 `x_session`。

#### session_table

==仅客户端==

生成会话 ID 使用的字符集，也可以是预定义表名。

`ALPHABET` `Alphabet` `BASE36` `Base62` `HEX` `alphabet` `base36` `hex` `number` `uuid` 之一，或任意 ASCII
字符串。

为空时生成随机 32 字符十六进制字符串，以兼容旧版本。

#### session_length

==仅客户端==

生成会话 ID 的长度范围，如 `16-32`。

默认使用 `16-32`。仅在设置了 `session_table` 时生效。

#### seq_placement

序列号的放置位置。

`path` `query` `header` `cookie` 之一。

默认使用 `path`。

#### seq_key

序列号位于查询参数、标头或 cookie 中时使用的参数名。

默认值取决于放置位置：标头为 `X-Seq`，cookie 和查询为 `x_seq`。

#### uplink_data_placement

packet-up 请求中上行负载的携带位置。

`body` `header` `cookie` `query` `auto` 之一。

默认使用 `body`。

#### uplink_data_key

上行数据位于标头或 cookie 中时的基础参数名，分块会附加序号后缀，如 `<key>-0`。无默认值。

#### uplink_chunk_size

上行数据位于标头或 cookie 中时的分块大小，范围字符串。

默认值取决于放置位置：cookie 为 `2048-3072`，标头为 `3072-4096`，否则为 `sc_max_each_post_bytes`。最小值为 64。

#### sc_max_each_post_bytes

packet-up 模式下单个 POST 请求体的最大大小，范围字符串。

默认使用 `1000000`。

#### sc_min_posts_interval_ms

两次 POST 请求之间的最小间隔，单位为毫秒，范围字符串。

默认使用 `30`。

#### reuse_settings

连接复用设置（即 Xray 的 `xmux`），用于 stream-up/stream-one 模式。

```yaml
reuse_settings:
  max_concurrency: ''
  max_connections: ''
  c_max_reuse_times: ''
  h_max_request_times: ''
  h_max_reusable_secs: ''
  h_keep_alive_period: 0
```

除 `h_keep_alive_period` 外，所有字段均为 `8-16` 形式的范围字符串或单个值，并按连接随机应用。为零则禁用对应限制。

##### max_concurrency

每个连接的最大并发流数。

##### max_connections

最大连接数。

##### c_max_reuse_times

单个连接可复用的最大次数。
##### h_max_request_times

每个连接的最大 HTTP 请求数。

##### h_max_reusable_secs

单个连接的最长生存时间，单位为秒。

##### h_keep_alive_period

HTTP/2 keep-alive 周期，单位为秒。

#### no_sse_header

==仅服务器==

默认情况下，stream-up/stream-one 响应会设置 `Content-Type: text/event-stream`，使 HTTP 中间设备禁用缓冲。

设置为 `true` 可省略该标头。

#### sc_stream_up_server_secs

==仅服务器==

服务器为保活 stream-up 连接而发送填充的时间间隔范围，单位为秒，如 `20-80`。

默认使用 `20-80`。为零则禁用保活填充。

#### sc_max_buffered_posts

packet-up 模式下每个会话缓冲的最大 POST 请求数。

默认使用 `30`。

#### download_settings

下载连接的专用设置，配合 `stream-up` 模式使用。

```yaml
download_settings:
  path: ''
  host: ''
  headers: {}
  reuse_settings: {}
  server: ''
  server_port: 0
  tls: {}
  alpn: []
```

`path`、`host`、`headers` 和 `reuse_settings` 的含义与对应的顶层字段相同。

`server` 和 `server_port` 覆盖下载服务器地址，`tls` 覆盖 TLS 配置，`alpn` 覆盖下载连接的 ALPN。

#### server_max_header_bytes

==仅服务器==

服务器接受的 HTTP 请求标头的最大大小，单位为字节。

默认使用 `8192`。小于或等于零的值将重置为默认值。
