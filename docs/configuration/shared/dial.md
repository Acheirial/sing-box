# Dial Fields

```{.yaml linenums="1"}
detour: ""
bind_interface: ""
inet4_bind_address: ""
inet6_bind_address: ""
bind_address_no_port: false
routing_mark: 0
reuse_addr: false
netns: ""
connect_timeout: ""
tcp_fast_open: false
tcp_multi_path: false
disable_tcp_keep_alive: false
tcp_keep_alive: ""
tcp_keep_alive_interval: ""
udp_fragment: false

domain_resolver: ""  # or {}
network_strategy: ""
network_type: []
fallback_network_type: []
fallback_delay: ""
```

!!! note ""

    You can use a single value instead of an array when the content is only one item

## detour

The tag of the upstream outbound.

If enabled, all other fields will be ignored.

## bind_interface

The network interface to bind to.

## inet4_bind_address

The IPv4 address to bind to.

## inet6_bind_address

The IPv6 address to bind to.

## bind_address_no_port

!!! quote ""

    Only supported on Linux.

Do not reserve a port when binding to a source address.

This allows reusing the same source port for multiple connections if the full 4-tuple (source IP, source port, destination IP, destination port) remains unique.

## routing_mark

!!! quote ""

    Only supported on Linux.

Set netfilter routing mark.

Integers (e.g. `1234`) and string hexadecimals (e.g. `"0x1234"`) are supported.

## reuse_addr

Reuse listener address.

## netns

!!! quote ""

    Only supported on Linux.

Set network namespace, name or path.

Since sing-box 1.14.0, the tag of a [network namespace](/configuration/network-namespace/) can also be used.
Referencing an `unshare` network namespace should be avoided.

## connect_timeout

Connect timeout, in golang's Duration format.

A duration string is a possibly signed sequence of
decimal numbers, each with optional fraction and a unit suffix,
such as "300ms", "-1.5h" or "2h45m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

## tcp_fast_open

Enable TCP Fast Open.

## tcp_multi_path

!!! warning ""

    Go 1.21 required.

Enable TCP Multi Path.

## disable_tcp_keep_alive

Disable TCP keep alive.

## tcp_keep_alive

TCP keep alive initial period.

`5m` will be used by default.

## tcp_keep_alive_interval

TCP keep alive interval.

`75s` will be used by default.

## udp_fragment

Enable UDP fragmentation.

## domain_resolver

!!! info ""

    `domain_resolver` or `route.default_domain_resolver` is optional when only one DNS server is configured.

Set domain resolver to use for resolving domain names.

This option uses the same format as the [route DNS rule action](/configuration/dns/rule_action/#route) without the `action` field.

Setting this option directly to a string is equivalent to setting `server` of this options.

| Outbound/Endpoints | Effected domains         |
|--------------------|--------------------------|
| `direct`           | Domain in request        | 
| others             | Domain in server address |

## network_strategy

!!! quote ""

    Only supported in graphical clients on Android and Apple platforms with `auto_detect_interface` enabled.

Strategy for selecting network interfaces.

Available values:

- `default` (default): Connect to default network or networks specified in `network_type` sequentially.
- `hybrid`: Connect to all networks or networks specified in `network_type` concurrently.
- `fallback`: Connect to default network or preferred networks specified in `network_type` concurrently, and try fallback networks when unavailable or timeout.

For fallback, when preferred interfaces fails or times out,
it will enter a 15s fast fallback state (Connect to all preferred and fallback networks concurrently),
and exit immediately if preferred networks recover.

Conflicts with `bind_interface`, `inet4_bind_address` and `inet6_bind_address`.

## network_type

!!! quote ""

    Only supported in graphical clients on Android and Apple platforms with `auto_detect_interface` enabled.

Network types to use when using `default` or `hybrid` network strategy or
preferred network types to use when using `fallback` network strategy.

Available values: `wifi`, `cellular`, `ethernet`, `other`.

Device's default network is used by default.

## fallback_network_type

!!! quote ""

    Only supported in graphical clients on Android and Apple platforms with `auto_detect_interface` enabled.

Fallback network types when preferred networks are unavailable or timeout when using `fallback` network strategy.

All other networks expect preferred are used by default.

## fallback_delay

!!! quote ""

    Only supported in graphical clients on Android and Apple platforms with `auto_detect_interface` enabled.

The length of time to wait before spawning a RFC 6555 Fast Fallback connection.

For `network_strategy`, is the amount of time to wait for connection to succeed before falling
back to other interfaces.

Only take effect when `network_strategy` is set.

`300ms` is used by default.
