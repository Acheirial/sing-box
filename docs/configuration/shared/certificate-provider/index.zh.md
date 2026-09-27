# 证书提供者

```{.yaml linenums="1"}
certificate_providers:
  - type: ""
    tag: ""
```

| 类型   | 格式             |
|--------|------------------|
| `acme` | [ACME](/zh/configuration/shared/certificate-provider/acme)   |
| `tailscale` | [Tailscale](/zh/configuration/shared/certificate-provider/tailscale) |
| `cloudflare-origin-ca` | [Cloudflare Origin CA](/zh/configuration/shared/certificate-provider/cloudflare-origin-ca) |

## tag

证书提供者的标签。
