---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# ShadowsocksR

### Structure

```yaml
type: shadowsocksr
tag: ssr-out
server: 127.0.0.1
server_port: 1080
method: aes-256-cfb
password: hello
obfs: plain
obfs_param: ""
protocol: origin
protocol_param: ""
network: tcp
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### method

==Required==

The encryption method.

One of:

* `rc4-md5`
* `aes-128-ctr`
* `aes-192-ctr`
* `aes-256-ctr`
* `aes-128-cfb`
* `aes-192-cfb`
* `aes-256-cfb`
* `chacha20`
* `chacha20-ietf`
* `xchacha20`
* `none`

`none` is an alias of the dummy cipher.

#### password

==Required==

The ShadowsocksR password.

#### obfs

Obfuscation method.

One of: `plain`, `http_simple`, `http_post`, `random_head`, `tls1.2_ticket_auth`, `tls1.2_ticket_fastauth`.

`plain` disables obfuscation.

#### obfs_param

Parameters for the obfuscation method.

#### protocol

ShadowsocksR protocol.

One of: `origin`, `auth_sha1_v4`, `auth_aes128_md5`, `auth_aes128_sha1`, `auth_chain_a`, `auth_chain_b`.

`origin` disables the protocol.

#### protocol_param

Parameters for the protocol.

#### network

Enabled network.

One of `tcp` `udp`.

Both are enabled by default.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
