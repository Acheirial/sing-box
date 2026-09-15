---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# Sudoku

### 结构

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

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### key

==必填==

用于派生加密种子和数表的共享密钥。

#### aead_method

AEAD 加密方法。

以下之一：`aes-128-gcm`、`chacha20-poly1305`、`none`。

默认使用 `chacha20-poly1305`。

#### padding_min

最小填充大小（字节）。

默认为 `10`。如果只设置 `padding_min`/`padding_max` 其中一个，另一个会被调整以保持范围一致。

#### padding_max

最大填充大小（字节）。

默认为 `30`。

#### table_type

数表生成模式，控制每个流量方向的首选线路布局。

以下之一：`prefer_ascii`、`prefer_entropy`、`up_ascii_down_ascii`、`up_ascii_down_entropy`、`up_entropy_down_ascii`、`up_entropy_down_entropy`。

默认使用 `prefer_entropy`。

#### enable_pure_downlink

启用纯下行优化。

默认为 `true`。

#### http_mask

启用手握流量的 HTTP 伪装。

#### http_mask_mode

HTTP 伪装隧道模式。

以下之一：`legacy`、`stream`、`poll`、`auto`、`ws`。

默认使用 `legacy`。

#### http_mask_tls

为 HTTP 伪装隧道启用 TLS。

#### http_mask_host

HTTP 伪装隧道的 Host 覆盖。

#### path_root

HTTP 伪装隧道的根路径。

#### multiplex

数据隧道的多路复用模式。

以下之一：`off`、`auto`、`on`。

默认使用 `off`。

#### http_mask_multiplex

HTTP 伪装隧道的多路复用模式覆盖。设置后同样作用于数据隧道。

#### httpmask

以单个对象形式提供的 HTTP 伪装选项。其中的字段覆盖顶层 HTTP 伪装字段。

| 字段        | 说明                                    |
|:-----------|:---------------------------------------|
| `disable`  | 禁用 HTTP 伪装                           |
| `mode`     | 隧道模式，同 `http_mask_mode`            |
| `tls`      | 启用 TLS，同 `http_mask_tls`             |
| `host`     | Host 覆盖，同 `http_mask_host`           |
| `path_root`| 根路径，同 `path_root`                   |
| `multiplex`| 多路复用模式，同 `multiplex`              |

#### custom_table

自定义数表模式。

#### custom_tables

自定义数表模式列表。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
