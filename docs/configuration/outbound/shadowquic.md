---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# ShadowQUIC

### Structure

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

### Fields

#### server

==Required==

The server address.

#### server_port

The server port. `443` is used by default.

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

Requires a standard TLS engine.

#### username

JLS authentication user name.

!!! note

    JLS authentication (username/password) is not supported in this build.

#### password

JLS authentication password.

#### alpn

TLS ALPN values. `h3` is used by default.

#### quic_versions

QUIC versions.

One of: `v1`, `v2`.

`v1` is used by default.

#### udp_over_stream

Enable the UDP over stream relay mode, which provides a QUIC stream based UDP
relay.

#### zero_rtt

Enable QUIC 0-RTT. DialEarly can only send 0-RTT data after TLS has cached a
session ticket from an earlier connection to this server.

#### keep_alive_interval

QUIC keep alive interval. Disabled when not set.

#### congestion_controller

QUIC congestion control algorithm.

One of: `cubic`, `new_reno`, `bbr_meta_v1`, `bbr_meta_v2`, `bbr`.

#### up

Upload bandwidth, used by the BBR congestion controller, e.g. `100 mbps`.

#### down

Download bandwidth, used by the BBR congestion controller, e.g. `100 mbps`.

#### cwnd

Initial congestion window. `32` is used by default.

#### bbr_profile

BBR profile.

One of: `standard`, `conservative`, `aggressive`.

#### receive_window_conn

QUIC stream receive window.

#### receive_window

QUIC connection receive window.

#### disable_mtu_discovery

Disable QUIC path MTU discovery.

#### max_datagram_frame_size

Maximum QUIC datagram frame size. `1400` is used by default.

#### max_open_streams

Maximum number of QUIC streams. `1024` is used by default.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
