---
icon: material/new-box
---

!!! question "Since sing-box 1.14.0"

# Default

Attach to an existing network namespace.

### Structure

```yaml
network_namespaces:
- type: default
  tag: ''
  path: ''
```

### Fields

#### path

==Required==

Name or path of the network namespace, for example `sing` or `/run/netns/sing`.
