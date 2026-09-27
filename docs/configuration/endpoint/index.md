# Endpoint

An endpoint is a protocol with inbound and outbound behavior.

```{.yaml linenums="1"}
endpoints:
  - type: ""
    tag: ""
```

| Type             | Format                                  |
|------------------|-----------------------------------------|
| `wireguard`      | [WireGuard](./wireguard/)               |
| `tailscale`      | [Tailscale](./tailscale/)               |
| `openconnect`    | [OpenConnect Client](./openconnect/)    |
| `openvpn-client` | [OpenVPN Client](./openvpn-client/)     |
| `openvpn-server` | [OpenVPN Server](./openvpn-server/)     |
| `masque-client`  | [MASQUE Client](./masque-client/)       |
| `masque-server`  | [MASQUE Server](./masque-server/)       |

## tag

The tag of the endpoint.
