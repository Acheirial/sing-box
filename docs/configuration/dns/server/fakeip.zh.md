# Fake IP

```{.yaml linenums="1"}
dns:
  servers:
    - type: fakeip
      tag: ""

      inet4_range: 198.18.0.0/15
      inet6_range: fc00::/18
```

## inet4_range

FakeIP 的 IPv4 地址范围。

## inet6_range

FakeIP 的 IPv6 地址范围。
