# UDP NAT 字段

```{.yaml linenums="1"}
udp_timeout: 5m
udp_mapping: endpoint_independent
udp_filtering: endpoint_independent
udp_nat_max: 0
```

## udp_timeout

UDP NAT 过期时间。

默认使用 `5m`。

## udp_mapping

UDP NAT 映射行为。

| 值                            | 行为                                                     |
|------------------------------|----------------------------------------------------------|
| `endpoint_independent`       | 对相同的源地址和端口，所有目标复用同一映射。                |
| `address_dependent`          | 每个目标地址使用单独的映射。                               |
| `address_and_port_dependent` | 每个目标地址和端口使用单独的映射。                          |

默认使用 `endpoint_independent`。

## udp_filtering

UDP NAT 过滤行为。

| 值                            | 行为                                                     |
|------------------------------|----------------------------------------------------------|
| `endpoint_independent`       | 接受来自任意远程端点的数据包。                              |
| `address_dependent`          | 仅接受来自已向其发送过数据包的远程地址的数据包。              |
| `address_and_port_dependent` | 仅接受来自已向其发送过数据包的远程地址和端口的数据包。         |

默认使用 `endpoint_independent`。

## udp_nat_max

UDP NAT 会话的最大数量。

达到限制时，将关闭最近最少使用的会话。

未设置或设置为 `0` 时，iOS 使用 `4096`。其他平台根据总内存在 `4096` 到 `16384` 之间选择；
无法检测总内存时使用 `16384`。
