# Outbound

### Structure

```yaml
outbounds:
- type: ''
  tag: ''
```

### Fields

| Type           | Format                         |
|----------------|--------------------------------|
| `direct`       | [Direct](./direct/)             |
| `bridge`       | [Bridge](./bridge/)             |
| `block`        | [Block](./block/)               |
| `socks`        | [SOCKS](./socks/)               |
| `http`         | [HTTP](./http/)                 |
| `shadowsocks`  | [Shadowsocks](./shadowsocks/)   |
| `vmess`        | [VMess](./vmess/)               |
| `trojan`       | [Trojan](./trojan/)             |
| `wireguard`    | [Wireguard](./wireguard/)       |
| `hysteria`     | [Hysteria](./hysteria/)         |
| `vless`        | [VLESS](./vless/)               |
| `shadowtls`    | [ShadowTLS](./shadowtls/)       |
| `tuic`         | [TUIC](./tuic/)                 |
| `hysteria2`    | [Hysteria2](./hysteria2/)       |
| `anytls`       | [AnyTLS](./anytls/)             |
| `snell`        | [Snell](./snell/)               |
| `tor`          | [Tor](./tor/)                   |
| `ssh`          | [SSH](./ssh/)                   |
| `dns`          | [DNS](./dns/)                   |
| `selector`     | [Selector](./selector/)         |
| `urltest`      | [URLTest](./urltest/)           |
| `naive`        | [NaiveProxy](./naive/)          |
| `shadowsocksr` | [ShadowsocksR](./shadowsocksr/) |
| `gost`         | [Gost](./gost/)                 |
| `mieru`        | [Mieru](./mieru/)               |
| `sudoku`       | [Sudoku](./sudoku/)             |
| `shadowquic`   | [ShadowQUIC](./shadowquic/)     |
| `masque`       | [MASQUE](./masque/)             |
| `trust-tunnel` | [TrustTunnel](./trust-tunnel/)  |
| `tls-mirror`   | [TLS Mirror](./tls-mirror/)     |
| `zerotier`     | [ZeroTier](./zerotier/)         |
| `easytier`     | [EasyTier](./easytier/)         |

#### tag

The tag of the outbound.

### Features

#### Outbounds that support IP connection

* `WireGuard`
