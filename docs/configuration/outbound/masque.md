---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# MASQUE

### Structure

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

### Fields

#### server

The server address. Port `443` is used by default.

#### server_port

The server port.

#### private_key

==Required==

The client private key.

#### public_key

==Required==

The server public key.

#### ip

Local IPv4 address assigned to the tunnel.

#### ipv6

Local IPv6 address assigned to the tunnel.

#### uri

MASQUE connect URI. `https://cloudflareaccess.com` is used by default.

#### mtu

Tunnel MTU. `1280` is used by default.

#### udp

Enable UDP support.

#### handshake_timeout

Handshake timeout in seconds. Disabled when not set. Must be non-negative.

#### skip_cert_verify

Skip TLS certificate verification.

#### network

MASQUE network mode.

One of: `h3`, `h3-l4proxy`.

`h3` tunnels connections through the userspace IP stack (requires the
`with_gvisor` build tag). `h3-l4proxy` forwards TCP connections directly and
requires resolved addresses.

#### congestion_controller

QUIC congestion control algorithm.

#### cwnd

Initial congestion window.

#### bbr_profile

BBR profile.

One of: `standard`, `conservative`, `aggressive`.

#### remote_dns_resolve

Resolve DNS queries through the tunnel.

#### dns

DNS servers used when `remote_dns_resolve` is enabled.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
