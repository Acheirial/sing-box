---
icon: material/new-box
---

# mDNS

!!! question "Since sing-box 1.14.0"

```{.yaml linenums="1"}
dns:
  servers:
    - type: mdns
      tag: ""

      interface: []

      # Dial Fields
```

!!! info ""

    `*.local.` and IPv4/IPv6 link-local reverse zones are also resolved by the [Local](./local/) server.

## interface

List of network interface names to send mDNS queries on.

When empty, all interfaces that are up, multicast-capable, and non-loopback are used.

## Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
