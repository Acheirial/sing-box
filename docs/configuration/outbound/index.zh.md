# 出站

### 结构

```yaml
outbounds:
- type: ''
  tag: ''
```

### 字段

| 类型             | 格式                             |
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

出站的标签。

### 特性

#### 支持 IP 连接的出站

* `WireGuard`
