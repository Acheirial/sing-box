---
icon: material/new-box
---

# Fake IP

!!! question "Since sing-box 1.12.0"

```{.yaml linenums="1"}
dns:
  servers:
    - type: fakeip
      tag: ""

      inet4_range: 198.18.0.0/15
      inet6_range: fc00::/18
```

## inet4_range

IPv4 address range for FakeIP.

## inet6_address

IPv6 address range for FakeIP.
