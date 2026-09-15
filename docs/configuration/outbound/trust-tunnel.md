---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# TrustTunnel

### Structure

```yaml
type: trust-tunnel
tag: trust-tunnel-out
server: 127.0.0.1
server_port: 443
tls: {}
username: ""
password: ""
udp: false
health_check: false
quic: false
congestion_controller: ""
cwnd: 0
bbr_profile: standard
max_connections: 0
min_streams: 0
max_streams: 0
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

ECH is incompatible with the uTLS engine.

#### username

Authentication user name.

#### password

Authentication password.

#### udp

Enable UDP support.

#### health_check

Enable periodic connection health checks.

#### quic

Use QUIC as the transport protocol instead of TCP.

#### congestion_controller

QUIC congestion control algorithm, used when `quic` is enabled.

#### cwnd

Initial congestion window, used when `quic` is enabled.

#### bbr_profile

BBR profile.

One of: `standard`, `conservative`, `aggressive`.

#### max_connections

Maximum number of pooled connections.

If all of `max_connections`, `min_streams` and `max_streams` are unset,
`max_connections` defaults to `8`.

#### min_streams

Minimum number of streams per connection before opening a new connection.

If all of `max_connections`, `min_streams` and `max_streams` are unset,
`min_streams` defaults to `5`.

#### max_streams

Maximum number of streams per connection.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
