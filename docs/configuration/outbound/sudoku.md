---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# Sudoku

### Structure

```yaml
type: sudoku
tag: sudoku-out
server: 127.0.0.1
server_port: 1080
key: hello
aead_method: chacha20-poly1305
padding_min: 10
padding_max: 30
table_type: prefer_entropy
enable_pure_downlink: true
http_mask: false
http_mask_mode: legacy
http_mask_tls: false
http_mask_host: ""
path_root: ""
multiplex: off
http_mask_multiplex: ""
httpmask:
  disable: false
  mode: ""
  tls: false
  host: ""
  path_root: ""
  multiplex: ""
custom_table: ""
custom_tables: []
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### key

==Required==

The shared key used to derive encryption seeds and tables.

#### aead_method

AEAD encryption method.

One of: `aes-128-gcm`, `chacha20-poly1305`, `none`.

`chacha20-poly1305` is used by default.

#### padding_min

Minimum padding size in bytes.

`10` is used by default. If only one of `padding_min`/`padding_max` is set,
the other side is adjusted to keep a consistent range.

#### padding_max

Maximum padding size in bytes.

`30` is used by default.

#### table_type

Table generation mode, which controls the preferred wire layout for each
traffic direction.

One of: `prefer_ascii`, `prefer_entropy`, `up_ascii_down_ascii`,
`up_ascii_down_entropy`, `up_entropy_down_ascii`, `up_entropy_down_entropy`.

`prefer_entropy` is used by default.

#### enable_pure_downlink

Enable the pure downlink optimization.

`true` is used by default.

#### http_mask

Enable HTTP masking of the handshake traffic.

#### http_mask_mode

HTTP mask tunnel mode.

One of: `legacy`, `stream`, `poll`, `auto`, `ws`.

`legacy` is used by default.

#### http_mask_tls

Enable TLS for the HTTP mask tunnel.

#### http_mask_host

Host override for the HTTP mask tunnel.

#### path_root

Root path for the HTTP mask tunnel.

#### multiplex

Multiplex mode for the data tunnel.

One of: `off`, `auto`, `on`.

`off` is used by default.

#### http_mask_multiplex

Multiplex mode override for the HTTP mask tunnel. Also applies to the data
tunnel when set.

#### httpmask

HTTP mask options as a single object. Fields inside override the top-level
HTTP mask fields.

| Field      | Description                        |
|:-----------|:-----------------------------------|
| `disable`  | Disable HTTP masking               |
| `mode`     | Tunnel mode, same as `http_mask_mode` |
| `tls`      | Enable TLS, same as `http_mask_tls` |
| `host`     | Host override, same as `http_mask_host` |
| `path_root`| Root path, same as `path_root`     |
| `multiplex`| Multiplex mode, same as `multiplex` |

#### custom_table

A custom table pattern.

#### custom_tables

A list of custom table patterns.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
