# Hysteria2

```{.yaml linenums="1"}
type: hysteria2
tag: hy2-in

# ... Listen Fields

up_mbps: 100
down_mbps: 100
obfs:
  type: salamander
  password: cry_me_a_r1ver
users:
  - name: tobyxdd
    password: goofy_ahh_password
ignore_client_bandwidth: false
disable_udp: false
tls: {}

# ... QUIC Fields

masquerade: ""  # or {}
bbr_profile: ""
brutal_debug: false
realm:
  server_url: https://realm.example.com
  token: ""
  realm_id: ""
  stun_servers: []
  stun_domain_resolver: ""  # or {}
  ip_version: 0
  port_mapping:
    enabled: false
    timeout: ""
    lifetime: ""
  http_client: {}
```

!!! warning "Difference from official Hysteria2"

    The official program supports an authentication method called **userpass**,
    which essentially uses a combination of `<username>:<password>` as the actual password,
    while sing-box does not provide this alias.
    To use sing-box with the official program, you need to fill in that combination as the actual password.

## Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

## up_mbps, down_mbps

Max bandwidth, in Mbps.

Not limited if empty.

Conflict with `ignore_client_bandwidth`.

## obfs.type

QUIC traffic obfuscator type, one of `salamander` `gecko`.

Disabled if empty.

## obfs.password

QUIC traffic obfuscator password.

## obfs.min_packet_size

Minimum on-wire packet size in bytes. Gecko only.

`512` is used by default.

## obfs.max_packet_size

Maximum on-wire packet size in bytes. Gecko only.

`1200` is used by default.

## users

Hysteria2 users

## users.password

Authentication password

## ignore_client_bandwidth

*When `up_mbps` and `down_mbps` are not set*:

Commands clients to use the BBR CC instead of Hysteria CC.

*When `up_mbps` and `down_mbps` are set*:

Deny clients to use the BBR CC.

## disable_udp

Disable UDP relay.

UDP is enabled by default.

## tls

**Required.** TLS configuration, see [TLS](/configuration/shared/tls/#inbound).

## QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

### masquerade

HTTP3 server behavior (URL string configuration) when authentication fails.

| Scheme       | Example                 | Description        |
|--------------|-------------------------|--------------------|
| `file`       | `file:///var/www`       | As a file server   |
| `http/https` | `http://127.0.0.1:8080` | As a reverse proxy |

Conflict with `masquerade.type`.

A 404 page will be returned if masquerade is not configured.

### masquerade.type

HTTP3 server behavior (Object configuration) when authentication fails.

| Type     | Description                 | Fields                              |
|----------|-----------------------------|-------------------------------------|
| `file`   | As a file server            | `directory`                         |
| `proxy`  | As a reverse proxy          | `url`, `rewrite_host`               |
| `string` | Reply with a fixed response | `status_code`, `headers`, `content` |

Conflict with `masquerade`.

A 404 page will be returned if masquerade is not configured.

### masquerade.directory

File server root directory.

### masquerade.url

Reverse proxy target URL.

### masquerade.rewrite_host

Rewrite the `Host` header to the target URL.

### masquerade.x_forwarded

Append the client address to the `X-Forwarded-For` header on proxied requests.

### masquerade.insecure

Disable verification of the reverse proxy target server certificate.

### masquerade.status_code

Fixed response status code.

### masquerade.headers

Fixed response headers.

### masquerade.content

Fixed response content.

### masquerade.listen_http

Start an additional plain HTTP listener on this address serving the same masquerade content.

Responses advertise the QUIC endpoint through the `Alt-Svc` header.

### masquerade.listen_https

Start an additional plain HTTPS listener on this address serving the same masquerade content, using the inbound TLS certificate.

Responses advertise the QUIC endpoint through the `Alt-Svc` header.

### masquerade.force_https

Redirect every request on `masquerade.listen_http` to `masquerade.listen_https` with `301 Moved Permanently`.

Requires both `masquerade.listen_http` and `masquerade.listen_https` to be set.

### bbr_profile

BBR congestion control algorithm profile, one of `conservative` `standard` `aggressive`.

`standard` is used by default.

### brutal_debug

Enable debug information logging for Hysteria Brutal CC.

### realm

Register this inbound to a Hysteria Realm rendezvous service to enable NAT traversal.

See [Hysteria Realm](/configuration/service/hysteria-realm/) for the rendezvous service.

### realm.server_url

**Required.** Realm rendezvous service URL.

### realm.token

Bearer token for the realm. Must match one of `users[].token` configured on the realm.

### realm.realm_id

**Required.** Slot identifier on the realm.

1–64 characters, must match `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`.

Outbounds must use the same `realm_id` to find this server.

### realm.stun_servers

**Required.** List of STUN servers (`host` or `host:port`) used to discover public addresses.

### realm.stun_domain_resolver

Set domain resolver to use for resolving STUN server domain names.

This option uses the same format as the [route DNS rule action](/configuration/dns/rule_action/#route) without the `action` field.

Setting this option directly to a string is equivalent to setting `server` of this options.

If empty, the default domain resolver is used.

### realm.ip_version

Restrict realm connections (STUN, hole punching, and the resulting QUIC path) to a single IP version.

`4` or `6`. Both are used if empty.

The `listen` address must be compatible with the selected version.

### realm.port_mapping

Maintain a UDP port mapping on the local gateway via UPnP or NAT-PMP.

Improves hole-punching reliability behind gateways that support it.

Requires IPv4: conflicts with `"ip_version": 6`.

### realm.port_mapping.enabled

Enable port mapping.

### realm.port_mapping.timeout

Timeout for gateway discovery and mapping operations.

`10s` is used by default.

### realm.port_mapping.lifetime

Lease lifetime of the mapping.

`10m` is used by default.

### realm.http_client

HTTP client used to talk to the realm.

See [HTTP Client](/configuration/shared/http-client/) for details.
