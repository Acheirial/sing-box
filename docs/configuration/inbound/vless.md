# VLESS

```{.yaml linenums="1"}
type: vless
tag: vless-in

decryption: ""

# ... Listen Fields

users:
  - name: sekai
    uuid: bf000d23-0752-40b4-affe-68f7707a9661
    flow: ""
tls: {}
multiplex: {}
transport: {}
```

## Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

## users

**Required.** VLESS users.

## users.uuid

**Required.** VLESS user id.

## users.flow

VLESS Sub-protocol.

Available values:

* `xtls-rprx-vision`

## decryption

VLESS Encryption, matching Xray's `decryption`.

When empty or `none`, the connection is untouched. Otherwise the value uses the
`mlkem768x25519plus` grammar, and the connection is decrypted before the VLESS
request is read:

```text
mlkem768x25519plus.<native|xorpub|random>.<from>[s]|[-<to>[s]][.<padding>][.<key>...]
```

* `native`, `xorpub` and `random` select the XOR mode applied to the relayed
  traffic.
* The second component is the ticket lifetime in seconds (a single value or a
  range such as `600s` or `600-1200s`).
* Optional following components shorter than 20 characters define padding, and
  the remaining ones are base64url-encoded keys (`xray vlessenc` output).

## tls

TLS configuration, see [TLS](/configuration/shared/tls/#inbound).

## multiplex

See [Multiplex](/configuration/shared/multiplex#inbound) for details.

## transport

Transport configuration, see [Transport](/configuration/shared/transport/).
