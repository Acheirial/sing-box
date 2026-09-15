### Structure

```yaml
type: vless
tag: vless-out
server: 127.0.0.1
server_port: 1080
uuid: bf000d23-0752-40b4-affe-68f7707a9661
flow: xtls-rprx-vision
network: tcp
tls: {}
packet_encoding: ''
encryption: ''
multiplex: {}
transport: {}
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### uuid

==Required==

VLESS user id.

#### flow

VLESS Sub-protocol.

Available values:

* `xtls-rprx-vision`

#### network

Enabled network

One of `tcp` `udp`.

Both is enabled by default.

#### encryption

!!! question "Since sing-box 1.15.0"

VLESS encryption, Xray's `mlkem768x25519plus` scheme.

Empty or `none` disables encryption.

Format: `mlkem768x25519plus.<mode>.<rtt>.<key>...`

* `mode`: One of `native` `xorpub` `random`.
* `rtt`: `1rtt` or `0rtt`. With `0rtt`, the client does not cache tickets.
* Trailing segments are base64 (RawURL) encoded keys (32-byte X25519 or 1184-byte ML-KEM-768 material) or short
  padding fragments.

The server counterpart is configured via the inbound `decryption` field. Setting `encryption` on inbound
`users[]` entries is an error.

#### tls

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

#### packet_encoding

UDP packet encoding, xudp is used by default.

| Encoding   | Description           |
|------------|-----------------------|
| (none)     | Disabled              |
| packetaddr | Supported by v2ray 5+ |
| xudp       | Supported by xray     |

#### multiplex

See [Multiplex](/configuration/shared/multiplex#outbound) for details.

#### transport

V2Ray Transport configuration, see [V2Ray Transport](/configuration/shared/v2ray-transport/).

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
