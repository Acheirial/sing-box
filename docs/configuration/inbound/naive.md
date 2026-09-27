# Naive

```{.yaml linenums="1"}
type: naive
tag: naive-in
network: udp

# ...

# Listen Fields

users:
  - username: sekai
    password: password
quic_congestion_control: ""
tls: {}
```

## Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

## network

Listen network, one of `tcp` `udp`.

Both if empty.

## users

**Required.** Naive users.

## quic_congestion_control

QUIC congestion control algorithm.

| Algorithm      | Description                     |
|----------------|---------------------------------|
| `bbr`          | BBR                             |
| `cubic`        | CUBIC                           |
| `reno`         | New Reno                        |

`bbr` is used by default.

## tls

TLS configuration, see [TLS](/configuration/shared/tls/#inbound).
