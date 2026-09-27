# DNS Server

```{.yaml linenums="1"}
dns:
  servers:
    - type: ""
      tag: ""
```

## type

DNS 服务器的类型。

| 类型              | 格式                        |
|-----------------|---------------------------|
| `local`         | [Local](./local/)         |
| `hosts`         | [Hosts](./hosts/)         |
| `tcp`           | [TCP](./tcp/)             |
| `udp`           | [UDP](./udp/)             |
| `tls`           | [TLS](./tls/)             |
| `quic`          | [QUIC](./quic/)           |
| `https`         | [HTTPS](./https/)         |
| `h3`            | [HTTP/3](./http3/)        |
| `dhcp`          | [DHCP](./dhcp/)           |
| `mdns`          | [mDNS](./mdns/)           |
| `fakeip`        | [Fake IP](./fakeip/)      |
| `tailscale`     | [Tailscale](./tailscale/) |
| `openconnect`   | [OpenConnect](./openconnect/) |
| `openvpn`       | [OpenVPN](./openvpn/)         |
| `resolved`      | [Resolved](./resolved/)   |

## tag

DNS 服务器的标签。
