# TProxy

!!! quote ""

    仅支持 Linux。

```{.yaml linenums="1"}
type: tproxy
tag: tproxy-in

# ... 监听字段

network: udp

# ... UDP NAT 字段

```

## 监听字段

参阅 [监听字段](/zh/configuration/shared/listen/)。

## network

监听的网络协议，`tcp` `udp` 之一。

默认所有。

## UDP NAT 字段

参阅 [UDP NAT 字段](/zh/configuration/shared/udp-nat/)。
