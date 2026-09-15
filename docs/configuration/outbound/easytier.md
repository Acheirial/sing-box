---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# EasyTier

### Structure

```yaml
type: easytier
tag: easytier-out
network_name: ""
network_secret: ""
hostname: ""
ipv4: ""
dhcp: false
peers: []
listeners: []
no_listener: false
mapped_listeners: []
exit_nodes: []
proxy_networks: []
instance_name: ""
state_directory: ""
udp: false
accept_dns: false
enable_exit_node: false
enable_encryption: false
encryption_algorithm: ""
private_mode: false
latency_first: false
```

### Fields

#### network_name

==Required==

The EasyTier network name.

#### network_secret

The EasyTier network secret.

#### hostname

The node host name.

#### ipv4

The local IPv4 address of the virtual network.

#### dhcp

Assign the local IPv4 address by DHCP. Enabled automatically when `ipv4` is
not set.

#### peers

Peer URIs to connect to, e.g. `tcp://public.easytier.top:11010`.

Required when `listeners` is empty.

#### listeners

Listener URIs.

#### no_listener

Disable listeners.

Conflicts with `listeners`.

#### mapped_listeners

Mapped listener URIs reported to the network.

#### exit_nodes

Exit node addresses.

#### proxy_networks

Network CIDRs to proxy through this node, e.g. `10.0.0.0/24`.

#### instance_name

The EasyTier instance name.

#### state_directory

The directory used to store EasyTier instance state. Defaults to a
tag-derived directory inside `easytier`.

#### udp

Enable UDP support.

#### accept_dns

Accept DNS requests through the virtual network.

#### enable_exit_node

Enable this node as an exit node.

#### enable_encryption

Enable P2P encryption.

#### encryption_algorithm

Encryption algorithm used when `enable_encryption` is enabled.

#### private_mode

Enable private mode, which refuses connections from unknown peers.

#### latency_first

Prefer the lowest-latency route instead of the default route selection.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
