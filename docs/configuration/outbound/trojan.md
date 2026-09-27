# Trojan

```{.yaml linenums="1"}
type: trojan
tag: trojan-out

server: 127.0.0.1
server_port: 1080
password: 8JCsPssfgS8tiRwiMlhARg==
network: tcp
tls: {}
multiplex: {}
transport: {}

# ... Dial Fields

```

## server

**Required.** The server address.

## server_port

**Required.** The server port.

## password

**Required.** The Trojan password.

## network

Enabled network

One of `tcp` `udp`.

Both is enabled by default.

## tls

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

## multiplex

See [Multiplex](/configuration/shared/multiplex#outbound) for details.

## transport

Transport configuration, see [Transport](/configuration/shared/transport/).

## Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
