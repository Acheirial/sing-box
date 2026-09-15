### 结构

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

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### password

==必填==

Trojan 密码。

#### network

启用的网络协议。

`tcp` 或 `udp`。

默认所有。

#### tls

TLS 配置, 参阅 [TLS](/zh/configuration/shared/tls/#出站)。

!!! question "自 sing-box 1.15.0 起"

支持可选的 `jls` 和 `restls` TLS 附加层。

两者均要求 `tls.enabled` 已启用，且配置在出站 TLS 选项中。
参阅 [TLS](/zh/configuration/shared/tls/#自定义-tls-支持)。

```yaml
tls:
  enabled: true
  server_name: example.org
  restls:
    password: pass
```

#### multiplex

参阅 [多路复用](/zh/configuration/shared/multiplex#出站)。

#### transport

V2Ray 传输配置，参阅 [V2Ray 传输层](/zh/configuration/shared/v2ray-transport/)。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
