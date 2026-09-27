---
icon: material/new-box
---

# DHCP

!!! question "自 sing-box 1.12.0 起"

```{.yaml linenums="1"}
dns:
  servers:
    - type: dhcp
      tag: ""

      interface: ""

      # 拨号字段
```

## interface

要监听的网络接口名称。

默认使用默认接口。

## 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/) 了解详情。
