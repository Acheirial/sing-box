---
icon: material/new-box
---

!!! question "Since sing-box 1.12.0"

# DHCP

### Structure

```yaml
dns:
  servers:
  - type: dhcp
    tag: ''
    interface: ''
```

### Fields

#### interface

Interface name to listen on. 

Tge default interface will be used by default.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details. 
