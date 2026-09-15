---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# Gost

gost relay protocol outbound.

### Structure

```yaml
type: gost
tag: gost-out
server: 127.0.0.1
server_port: 1080
tls: {}
forward: false
udp: false
mux: false
username: ""
password: ""
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### tls

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

#### forward

Enable forward mode. The server address is only used as the relay address and
the actual destination is passed to the relay server.

If disabled, the connection to the relay server is established per destination
and the destination address is carried in the relay request.

#### udp

Enable UDP support. UDP packets are relayed with the UDP over TCP protocol.

#### mux

Enable multiplexing, see [Multiplex](/configuration/shared/multiplex/) for details.

#### username

Relay authentication user name.

#### password

Relay authentication password.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
