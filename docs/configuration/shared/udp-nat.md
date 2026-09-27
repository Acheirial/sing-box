# UDP NAT Fields

```{.yaml linenums="1"}
udp_timeout: 5m
udp_mapping: endpoint_independent
udp_filtering: endpoint_independent
udp_nat_max: 0
```

## udp_timeout

UDP NAT expiration time.

`5m` will be used by default.

## udp_mapping

UDP NAT mapping behavior.

| Value                        | Behavior                                                                      |
|------------------------------|-------------------------------------------------------------------------------|
| `endpoint_independent`       | Reuse the same mapping for the same source address and port for all destinations. |
| `address_dependent`          | Use a separate mapping for each destination address.                          |
| `address_and_port_dependent` | Use a separate mapping for each destination address and port.                 |

`endpoint_independent` is used by default.

## udp_filtering

UDP NAT filtering behavior.

| Value                        | Behavior                                                                    |
|------------------------------|-----------------------------------------------------------------------------|
| `endpoint_independent`       | Accept packets from any remote endpoint.                                    |
| `address_dependent`          | Accept packets only from remote addresses to which packets have been sent.  |
| `address_and_port_dependent` | Accept packets only from remote addresses and ports to which packets have been sent. |

`endpoint_independent` is used by default.

## udp_nat_max

Maximum number of UDP NAT sessions.

When the limit is reached, the least recently used session is closed.

When unset or set to `0`, `4096` is used on iOS. On other platforms, a value from `4096` to `16384` is selected based on total memory;
`16384` is used if total memory cannot be detected.
