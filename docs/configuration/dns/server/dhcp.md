---
icon: material/new-box
---

# DHCP

!!! question "Since sing-box 1.12.0"

```{.yaml linenums="1"}
dns:
  servers:
    - type: dhcp
      tag: ""

      interface: ""

      # Dial Fields
```

## interface

Interface name to listen on. 

Tge default interface will be used by default.

## Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details. 
