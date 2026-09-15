---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# ZeroTier

### 结构

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

### 字段

#### network

==必填==

ZeroTier 网络 ID。

#### state_dir

存储 ZeroTier 节点状态的目录。默认为 `zerotier` 下由标签派生的目录。

#### identity_secret

节点身份密钥。未设置时生成新身份。

#### planet

自定义 planet 文件路径。

#### mtu

网络 MTU。默认使用网络分配的 MTU。

#### ip_stack

IP 栈模式。

以下之一：`auto`、`system`、`gvisor`、`mixed`。

默认使用 `auto`：启用 `with_gvisor` 构建标签时使用 `gvisor`，否则使用 `system`。

#### physical_mtu

物理接口 MTU。

#### udp

启用 UDP 支持。

#### remote_dns_resolve

使用 `dns` 中的 DNS 服务器通过虚拟网络解析 DNS 查询。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
