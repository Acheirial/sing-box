---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# TLS Mirror

### Structure

```yaml
type: tls-mirror
tag: tls-mirror-out
server: 127.0.0.1
server_port: 443
tls: {}
primary_key: ""
explicit_nonce_ciphersuites: []
defer_instance_derived_write_time:
  base_nanoseconds: 0
  uniform_random_multiplier_nanoseconds: 0
transport_layer_padding:
  enabled: false
connection_enrolment:
  primary_ingress_outbound: ""
  primary_egress_outbound: ""
embedded_traffic_generator:
  steps:
    - name: ""
      host: ""
      path: ""
      method: ""
      headers:
        - name: ""
          value: ""
          values: []
      next_step:
        - weight: 0
          goto_location: 0
      connection_ready: false
      connection_recall_exit: false
      wait_time:
        base_nanoseconds: 0
        uniform_random_multiplier_nanoseconds: 0
      h2_do_not_wait_for_download_finish: false
sequence_watermarking_enabled: false
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

#### primary_key

==Required==

The primary key used to derive mirroring state.

#### primary_ingress_outbound

Tag of the outbound used as the primary ingress connection for connection
enrolment. Used inside `connection_enrolment`; at least one of
`primary_ingress_outbound` or `primary_egress_outbound` is required when
enrolment is enabled, and the referenced outbound cannot be the TLS Mirror
outbound itself.

#### primary_egress_outbound

Tag of the outbound used as the primary egress connection for connection
enrolment. Falls back to `primary_ingress_outbound` when empty.

#### explicit_nonce_ciphersuites

TLS 1.2 cipher suite IDs that carry explicit nonces, as a list of integers.

#### defer_instance_derived_write_time

Timing specification for deferring instance derived writes.

| Field                                  | Description                                      |
|:---------------------------------------|:-------------------------------------------------|
| `base_nanoseconds`                     | Base delay in nanoseconds                        |
| `uniform_random_multiplier_nanoseconds`| Uniform random multiplier added to the base delay, in nanoseconds |

#### transport_layer_padding

Transport layer padding options.

| Field     | Description               |
|:----------|:--------------------------|
| `enabled` | Enable transport layer padding |

#### connection_enrolment

Connection enrolment options, see `primary_ingress_outbound` and
`primary_egress_outbound`.

#### embedded_traffic_generator

Embedded traffic generator used to shape the appearance of the connection.

Each step supports:

| Field                                  | Description                                    |
|:---------------------------------------|:-----------------------------------------------|
| `name`                                 | Step name                                      |
| `host`                                 | Request host                                   |
| `path`                                 | Request path                                   |
| `method`                               | Request method                                 |
| `headers`                              | Request headers, each with `name` and `value` or `values` |
| `next_step`                            | Transfer candidates, each with `weight` and `goto_location` |
| `connection_ready`                     | Mark the connection as ready                    |
| `connection_recall_exit`               | Exit after recalling the connection             |
| `wait_time`                            | Wait timing specification, same as `defer_instance_derived_write_time` |
| `h2_do_not_wait_for_download_finish`   | Do not wait for the HTTP/2 download to finish   |

#### sequence_watermarking_enabled

Enable sequence number watermarking.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
