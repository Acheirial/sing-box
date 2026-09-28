# VLESS

```{.yaml linenums="1"}
type: vless
tag: vless-out

server: 127.0.0.1
server_port: 1080
uuid: bf000d23-0752-40b4-affe-68f7707a9661
flow: xtls-rprx-vision
vless_route: 0
encryption: ""
network: tcp
tls: {}
packet_encoding: ""
multiplex: {}
transport: {}

# ... Dial Fields

```

## server

**Required.** The server address.

## server_port

**Required.** The server port.

## uuid

**Required.** VLESS user id.

## flow

VLESS Sub-protocol.

Available values:

* `xtls-rprx-vision`

## vless_route

Integer routing tag encoded into bytes 6 and 7 of the user UUID, matching Xray vlessRoute.

## encryption

VLESS Encryption, matching Xray's user `encryption`.

When empty or `none`, the connection is untouched. Otherwise the value uses the
`mlkem768x25519plus` grammar, and the connection is encrypted before the VLESS
request is written:

```text
mlkem768x25519plus.<native|xorpub|random>.<1rtt|0rtt>[.<padding>][.<key>...]
```

* `native`, `xorpub` and `random` select the XOR mode applied to the relayed
  traffic.
* `1rtt` disables session tickets, `0rtt` reuses them.
* Optional following components shorter than 20 characters define padding, and
  the remaining ones are base64url-encoded keys (`xray vlessenc` output).

The matching inbound `decryption` value must be generated from the same handshake.

## network

Enabled network

One of `tcp` `udp`.

Both is enabled by default.

## tls

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

## packet_encoding

UDP packet encoding, xudp is used by default.

| Encoding   | Description           |
|------------|-----------------------|
| (none)     | Disabled              |
| packetaddr | Supported by v2ray 5+ |
| xudp       | Supported by xray     |

## multiplex

See [Multiplex](/configuration/shared/multiplex#outbound) for details.

## transport

Transport configuration, see [Transport](/configuration/shared/transport/).

## Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
