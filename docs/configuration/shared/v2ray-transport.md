V2Ray Transport is a set of private protocols invented by v2ray, and has contaminated the names of other protocols, such
as `trojan-grpc` in clash.

### Structure

```yaml
type: ''
```

Available transports:

* HTTP
* WebSocket
* QUIC
* gRPC
* HTTPUpgrade
* XHTTP

!!! warning "Difference from v2ray-core"

    * No TCP transport, plain HTTP is merged into the HTTP transport.
    * No mKCP transport.
    * No DomainSocket transport.

!!! note ""

    You can ignore the JSON Array [] tag when the content is only one item

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

!!! warning "Difference from v2ray-core"

    TLS is not enforced. If TLS is not configured, plain HTTP 1.1 is used.

#### host

List of host domain.

The client will choose randomly and the server will verify if not empty.

#### path

!!! warning

    V2Ray's documentation says that the path between the server and the client must be consistent, 
    but the actual code allows the client to add any suffix to the path.
    sing-box uses the same behavior as V2Ray, but note that the behavior does not exist in `WebSocket` and `HTTPUpgrade` transport.

Path of HTTP request.

The server will verify.

#### method

Method of HTTP request.

The server will verify if not empty.

#### headers

Extra headers of HTTP request.

The server will write in response if not empty.

#### idle_timeout

In HTTP2 server:

Specifies the time until idle clients should be closed with a GOAWAY frame. PING frames are not considered as activity.

In HTTP2 client:

Specifies the period of time after which a health check will be performed using a ping frame if no frames have been
received on the connection.Please note that a ping response is considered a received frame, so if there is no other
traffic on the connection, the health check will be executed every interval. If the value is zero, no health check will
be performed.

Zero is used by default.

#### ping_timeout

In HTTP2 client:

Specifies the timeout duration after sending a PING frame, within which a response must be received.
If a response to the PING frame is not received within the specified timeout duration, the connection will be closed.
The default timeout duration is 15 seconds.

### WebSocket

```yaml
type: ws
path: ''
headers: {}
max_early_data: 0
early_data_header_name: ''
```

#### path

Path of HTTP request.

The server will verify.

#### headers

Extra headers of HTTP request.

The server will write in response if not empty.

#### max_early_data

Allowed payload size is in the request. Enabled if not zero.

#### early_data_header_name

Early data is sent in path instead of header by default.

To be compatible with Xray-core, set this to `Sec-WebSocket-Protocol`.

It needs to be consistent with the server.

### QUIC

```yaml
type: quic
```

!!! warning "Difference from v2ray-core"

    No additional encryption support:
    It's basically duplicate encryption. And Xray-core is not compatible with v2ray-core in here.

### gRPC

!!! note ""

    standard gRPC has good compatibility but poor performance and is not included by default, see [Installation](/installation/build-from-source/#build-tags).

```yaml
type: grpc
service_name: TunService
idle_timeout: 15s
ping_timeout: 15s
permit_without_stream: false
```

#### service_name

Service name of gRPC.

#### idle_timeout

In standard gRPC server/client:

If the transport doesn't see any activity after a duration of this time,
it pings the client to check if the connection is still active.

In default gRPC server/client:

It has the same behavior as the corresponding setting in HTTP transport.

#### ping_timeout

In standard gRPC server/client:

The timeout that after performing a keepalive check, the client will wait for activity.
If no activity is detected, the connection will be closed.

In default gRPC server/client:

It has the same behavior as the corresponding setting in HTTP transport.

#### permit_without_stream

In standard gRPC client:

If enabled, the client transport sends keepalive pings even with no active connections.
If disabled, when there are no active connections, `idle_timeout` and `ping_timeout` will be ignored and no keepalive
pings will be sent.

Disabled by default.

### HTTPUpgrade

```yaml
type: httpupgrade
host: ''
path: ''
headers: {}
```

#### host

Host domain.

The server will verify if not empty.

#### path

Path of HTTP request.

The server will verify.

#### headers

Extra headers of HTTP request.

The server will write in response if not empty.

### XHTTP

!!! question "Since sing-box 1.15.0"

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

Host domain.

#### path

Path of HTTP request.

#### mode

XHTTP transport mode.

One of `auto` `packet-up` `stream-up` `stream-one`.

`auto` is used by default. With `auto`, `stream-one` is used with REALITY, `stream-up` with REALITY and
`download_settings`, otherwise `packet-up`.

#### extra

An overlay object applied over the transport configuration after parsing. Fields set in `extra` override the
corresponding top-level fields, providing forward compatibility with newer Xray options.

`host`, `path` and `mode` are immune to override. Unknown keys are ignored.

#### headers

Extra headers of HTTP requests.

#### alpn

ALPN list for the HTTP transport.

Defaults to the ALPN of the TLS configuration when not set.

#### no_grpc_header

By default, stream-up/stream-one requests with a body are marked with a `Content-Type: application/grpc` header.

Set to `true` to omit it.

#### x_padding_bytes

Padding length, a range string like `100-1000` or a single value.

`100-1000` is used by default.

#### x_padding_obfs_mode

Enable the obfuscated padding mode with configurable placement, key, header and method.

When disabled, padding uses the legacy format: a query parameter named `x_padding` in the `Referer` header of
client requests, and an `X-Padding` response header from the server.

#### x_padding_key

Parameter name of the padding when placed in a cookie or query.

The legacy padding format uses `x_padding`.

#### x_padding_header

Header name of the padding when placed in a header.

#### x_padding_placement

Where the padding is placed.

One of `header` `queryInHeader` `cookie` `query`.

#### x_padding_method

Padding generation method.

One of `repeat-x` `tokenish`.

`repeat-x` fills the padding with repeated `X` characters; `tokenish` generates random base62 strings that look
like tokens. If empty, `repeat-x` is used.

#### uplink_http_method

HTTP method used for uplink requests.

`POST` is used by default.

#### session_placement

Where the session ID is placed.

One of `path` `query` `header` `cookie`.

`path` is used by default.

#### session_key

Parameter name of the session ID when placed in a query, header or cookie.

Defaults depend on the placement: `X-Session` for headers, `x_session` for cookies and queries.

#### session_table

==Client only==

Character set used to generate session IDs. Can also be a predefined table name.

One of `ALPHABET` `Alphabet` `BASE36` `Base62` `HEX` `alphabet` `base36` `hex` `number` `uuid`, or an arbitrary
ASCII string.

If empty, a random 32-character hexadecimal string is generated for compatibility with older versions.

#### session_length

==Client only==

Length range of generated session IDs, e.g. `16-32`.

`16-32` is used by default. Only takes effect when `session_table` is set.

#### seq_placement

Where the sequence number is placed.

One of `path` `query` `header` `cookie`.

`path` is used by default.

#### seq_key

Parameter name of the sequence number when placed in a query, header or cookie.

Defaults depend on the placement: `X-Seq` for headers, `x_seq` for cookies and queries.

#### uplink_data_placement

Where the uplink payload is carried in packet-up requests.

One of `body` `header` `cookie` `query` `auto`.

`body` is used by default.

#### uplink_data_key

Base parameter name of uplink data chunks when placed in headers or cookies. Chunks are suffixed with their
index, e.g. `<key>-0`. No default value is applied.

#### uplink_chunk_size

Chunk size of uplink data carried in headers or cookies, a range string.

Defaults depend on the placement: `2048-3072` for cookies, `3072-4096` for headers, otherwise
`sc_max_each_post_bytes`. The minimum is 64.

#### sc_max_each_post_bytes

Maximum size of each POST request body in packet-up mode, a range string.

`1000000` is used by default.

#### sc_min_posts_interval_ms

Minimum interval between two POST requests, in milliseconds, a range string.

`30` is used by default.

#### reuse_settings

Connection reuse settings (Xray's `xmux`), used in stream-up/stream-one modes.

```yaml
reuse_settings:
  max_concurrency: ''
  max_connections: ''
  c_max_reuse_times: ''
  h_max_request_times: ''
  h_max_reusable_secs: ''
  h_keep_alive_period: 0
```

All fields except `h_keep_alive_period` are range strings like `8-16` or single values, and are randomly applied
per connection. Zero disables the corresponding limit.

##### max_concurrency

Maximum concurrent streams per connection.

##### max_connections

Maximum number of connections.

##### c_max_reuse_times

Maximum times a connection can be reused.

##### h_max_request_times

Maximum number of HTTP requests per connection.

##### h_max_reusable_secs

Maximum lifetime of a connection, in seconds.

##### h_keep_alive_period

HTTP/2 keep-alive period, in seconds.

#### no_sse_header

==Server only==

By default, stream-up/stream-one responses set `Content-Type: text/event-stream` so that HTTP middleboxes
disable buffering.

Set to `true` to omit it.

#### sc_stream_up_server_secs

==Server only==

Interval range, in seconds, at which the server sends padding to keep stream-up connections alive, e.g. `20-80`.

`20-80` is used by default. Zero disables keep-alive padding.

#### sc_max_buffered_posts

==Server only==

Maximum number of buffered POST requests per session in packet-up mode.

`30` is used by default.

#### download_settings

Dedicated settings for the download connection, used with `stream-up` mode.

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

`path`, `host`, `headers` and `reuse_settings` have the same meaning as the corresponding top-level fields.

`server` and `server_port` override the download server address, `tls` overrides the TLS configuration, and
`alpn` overrides the ALPN for the download connection.

#### server_max_header_bytes

==Server only==

The maximum size of HTTP request headers accepted by the server, in bytes.

`8192` is used by default. Values less than or equal to zero reset to the default.
