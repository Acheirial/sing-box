# Default

附加到已存在的网络命名空间。

```{.yaml linenums="1"}
network_namespaces:
  - type: default  # 可选
    tag: ""
    path: ""
```

## path

**必填。**网络命名空间的名称或路径，例如 `sing` 或 `/run/netns/sing`。
