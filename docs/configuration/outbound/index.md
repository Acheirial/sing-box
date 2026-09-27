# Outbound

```{.yaml linenums="1"}
outbounds:
  - type: ""
    tag: ""
```

| Type           | Format                         |
|----------------|--------------------------------|
| `direct`       | [Direct](./direct/)             |
| `bridge`       | [Bridge](./bridge/)             |
| `socks`        | [SOCKS](./socks/)               |
| `http`         | [HTTP](./http/)                 |
| `shadowsocks`  | [Shadowsocks](./shadowsocks/)   |
| `vmess`        | [VMess](./vmess/)               |
| `trojan`       | [Trojan](./trojan/)             |
| `hysteria`     | [Hysteria](./hysteria/)         |
| `vless`        | [VLESS](./vless/)               |
| `shadowtls`    | [ShadowTLS](./shadowtls/)       |
| `tuic`         | [TUIC](./tuic/)                 |
| `hysteria2`    | [Hysteria2](./hysteria2/)       |
| `anytls`       | [AnyTLS](./anytls/)             |
| `snell`        | [Snell](./snell/)               |
| `tailcat`      | [Tailcat](./tailcat/)           |
| `tor`          | [Tor](./tor/)                   |
| `ssh`          | [SSH](./ssh/)                   |
| `selector`     | [Selector](./selector/)         |
| `urltest`      | [URLTest](./urltest/)           |
| `naive`        | [NaiveProxy](./naive/)          |

## tag

The tag of the outbound.

## Features

### Outbounds that support IP connection

* `WireGuard`
