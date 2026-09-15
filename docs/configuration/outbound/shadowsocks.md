### Structure

```yaml
type: shadowsocks
tag: ss-out
server: 127.0.0.1
server_port: 1080
method: 2022-blake3-aes-128-gcm
password: 8JCsPssfgS8tiRwiMlhARg==
plugin: ''
plugin_opts: ''
network: udp
udp_over_tcp: false
multiplex: {}
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

Encryption methods:

* `2022-blake3-aes-128-gcm`
* `2022-blake3-aes-256-gcm`
* `2022-blake3-chacha20-poly1305`
* `none`
* `aes-128-gcm`
* `aes-192-gcm`
* `aes-256-gcm`
* `chacha20-ietf-poly1305`
* `xchacha20-ietf-poly1305`

Legacy encryption methods:

* `aes-128-ctr`
* `aes-192-ctr`
* `aes-256-ctr`
* `aes-128-cfb`
* `aes-192-cfb`
* `aes-256-cfb`
* `rc4-md5`
* `chacha20-ietf`
* `xchacha20`

#### password

==Required==

The shadowsocks password.

#### plugin

Shadowsocks SIP003 plugin, implemented internally.

Supported plugins:

* `obfs-local`
* `v2ray-plugin`
* `restls`
* `jls`
* `kcptun`

For the `restls` plugin, the following options are supported in `plugin_opts`:
`password`, `version_hint` (`tls12` or `tls13`, defaults to `tls13`),
`restls_script`, plus optional `skip_cert_verify`, `force_tls12`,
`fingerprint`, `name_cert_verify` and `host`.

#### plugin_opts

Shadowsocks SIP003 plugin options.

#### network

Enabled network

One of `tcp` `udp`.

Both is enabled by default.

#### udp_over_tcp

UDP over TCP configuration.

See [UDP Over TCP](/configuration/shared/udp-over-tcp/) for details.

Conflict with `multiplex`.

#### multiplex

See [Multiplex](/configuration/shared/multiplex#outbound) for details.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
