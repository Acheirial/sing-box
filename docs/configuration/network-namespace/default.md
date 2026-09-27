# Default

Attach to an existing network namespace.

```{.yaml linenums="1"}
network_namespaces:
  - type: default  # optional
    tag: ""
    path: ""
```

## path

**Required.** Name or path of the network namespace, for example `sing` or `/run/netns/sing`.
