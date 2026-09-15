---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# EasyTier

### 结构

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

### 字段

#### network_name

==必填==

EasyTier 网络名称。

#### network_secret

EasyTier 网络密钥。

#### hostname

节点主机名。

#### ipv4

虚拟网络的本地 IPv4 地址。

#### dhcp

通过 DHCP 分配本地 IPv4 地址。未设置 `ipv4` 时自动启用。

#### peers

要连接的对端 URI，例如 `tcp://public.easytier.top:11010`。

`listeners` 为空时必填。

#### listeners

监听 URI 列表。

#### no_listener

禁用监听。

与 `listeners` 冲突。

#### mapped_listeners

向网络报告的映射监听 URI 列表。

#### exit_nodes

出口节点地址列表。

#### proxy_networks

通过本节点代理的网络 CIDR，例如 `10.0.0.0/24`。

#### instance_name

EasyTier 实例名称。

#### state_directory

存储 EasyTier 实例状态的目录。默认为 `easytier` 下由标签派生的目录。

#### udp

启用 UDP 支持。

#### accept_dns

接受通过虚拟网络的 DNS 请求。

#### enable_exit_node

启用本节点作为出口节点。

#### enable_encryption

启用 P2P 加密。

#### encryption_algorithm

启用 `enable_encryption` 时使用的加密算法。

#### private_mode

启用私有模式，拒绝来自未知对端的连接。

#### latency_first

优先选择延迟最低的路由，而不是默认的路由选择。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
