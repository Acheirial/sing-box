### Structure

```yaml
type: trojan
tag: trojan-out
server: 127.0.0.1
server_port: 1080
password: 8JCsPssfgS8tiRwiMlhARg==
network: tcp
tls: {}
multiplex: {}
transport: {}
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### password

==Required==

The Trojan password.

#### network

Enabled network

One of `tcp` `udp`.

Both is enabled by default.

#### tls

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

!!! question "Since sing-box 1.15.0"

Supports the optional `jls` and `restls` TLS add-on layers.

Both layers require `tls.enabled` and are configured in the outbound TLS
options. See [TLS](/configuration/shared/tls/#custom-tls-support) for details.

```yaml
tls:
  enabled: true
  server_name: example.org
  restls:
    password: pass
```

#### multiplex

See [Multiplex](/configuration/shared/multiplex#outbound) for details.

#### transport

V2Ray Transport configuration, see [V2Ray Transport](/configuration/shared/v2ray-transport/).

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
