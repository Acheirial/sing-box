---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# ZeroTier

### Structure

```yaml
type: zerotier
tag: zerotier-out
network: ""
state_dir: ""
identity_secret: ""
planet: ""
mtu: 0
ip_stack: auto
physical_mtu: 0
udp: false
remote_dns_resolve: false
```

### Fields

#### network

==Required==

The ZeroTier network ID.

#### state_dir

The directory used to store ZeroTier node state. Defaults to a tag-derived
directory inside `zerotier`.

#### identity_secret

The node identity secret. A new identity is generated when not set.

#### planet

Path to a custom planet file.

#### mtu

Network MTU. Defaults to the MTU assigned by the network.

#### ip_stack

IP stack mode.

One of: `auto`, `system`, `gvisor`, `mixed`.

`auto` is used by default: `gvisor` is used when the `with_gvisor` build tag
is enabled, otherwise `system`.

#### physical_mtu

Physical interface MTU.

#### udp

Enable UDP support.

#### remote_dns_resolve

Resolve DNS queries through the virtual network using the DNS servers in `dns`.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
