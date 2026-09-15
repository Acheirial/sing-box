---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# Mieru

### Structure

```yaml
type: mieru
tag: mieru-out
server: 127.0.0.1
server_port: 1080
port_range: ""
transport: tcp
udp: false
username: ""
password: ""
multiplexing: ""
handshake_mode: ""
traffic_pattern: ""
```

### Fields

#### server

==Required==

The server address.

#### server_port

The server port.

Conflicts with `port_range`. Either `server_port` or `port_range` must be set.

#### port_range

The server port range, in `begin:end` format, e.g. `10000:20000`.

Conflicts with `server_port`. Either `server_port` or `port_range` must be set.

#### transport

Transport protocol.

One of: `tcp`, `udp`.

#### udp

Enable UDP support.

#### username

User name.

#### password

Password.

#### multiplexing

Multiplexing level.

One of: `MULTIPLEXING_DEFAULT`, `MULTIPLEXING_OFF`, `MULTIPLEXING_LOW`, `MULTIPLEXING_MIDDLE`, `MULTIPLEXING_HIGH`.

#### handshake_mode

Handshake mode.

One of: `HANDSHAKE_DEFAULT`, `HANDSHAKE_STANDARD`, `HANDSHAKE_NO_WAIT`.

`HANDSHAKE_STANDARD` is also known as 1-RTT: the client waits for the proxy
server to establish the connection to the destination before sending the
payload. `HANDSHAKE_NO_WAIT` is also known as 0-RTT: the client sends the
payload at the same time when connecting to the proxy server.

#### traffic_pattern

Traffic pattern, encoded as a base64 string.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
