# Certificate Provider

```{.yaml linenums="1"}
certificate_providers:
  - type: ""
    tag: ""
```

| Type   | Format           |
|--------|------------------|
| `acme` | [ACME](/configuration/shared/certificate-provider/acme)   |
| `tailscale` | [Tailscale](/configuration/shared/certificate-provider/tailscale) |
| `cloudflare-origin-ca` | [Cloudflare Origin CA](/configuration/shared/certificate-provider/cloudflare-origin-ca) |

## tag

The tag of the certificate provider.
