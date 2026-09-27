# DNS 规则

```{.yaml linenums="1"}
dns:
  rules:
    - inbound:
        - mixed-in
      ip_version: 6
      query_type:
        - A
        - HTTPS
        - 32768
      query_client_subnet:
        - 10.0.0.0/24
        - 192.168.0.1
      query_dnssec: false
      network: tcp
      auth_user:
        - usera
        - userb
      protocol:
        - tls
        - http
        - quic
      domain:
        - test.com
      domain_suffix:
        - .cn
      domain_keyword:
        - test
      domain_regex:
        - ^stun\..+
      source_ip_cidr:
        - 10.0.0.0/24
        - 192.168.0.1
      source_ip_is_private: false
      source_port:
        - 12345
      source_port_range:
        - "1000:2000"
        - ":3000"
        - "4000:"
      port:
        - 80
        - 443
      port_range:
        - "1000:2000"
        - ":3000"
        - "4000:"
      process_name:
        - curl
      process_path:
        - /usr/bin/curl
      process_path_regex:
        - ^/usr/bin/.+
      package_name:
        - com.termux
      package_name_regex:
        - ^com\.termux.*
      user:
        - sekai
      user_id:
        - 1000
      clash_mode: direct
      network_type:
        - wifi
      network_is_expensive: false
      network_is_constrained: false
      interface_address:
        en0:
          - 2000::/3
      network_interface_address:
        wifi:
          - 2000::/3
      default_interface_address:
        - 2000::/3
      source_mac_address:
        - "00:11:22:33:44:55"
      source_hostname:
        - my-device
      preferred_by:
        - local
        - ts-dns
      dns_server_address:
        local:
          - 192.168.1.1/32
      dns_search_domain:
        ts-dns:
          - example.ts.net
      wifi_ssid:
        - My WIFI
      wifi_bssid:
        - "00:00:00:00:00:00"
      rule_set:
        - geoip-cn
        - geosite-cn
      rule_set_ip_cidr_match_source: false
      match_response: false
      ip_cidr:
        - 10.0.0.0/24
        - 192.168.0.1
      ip_is_private: false
      ip_accept_any: false
      response_rcode: ""
      response_answer: []
      response_ns: []
      response_extra: []
      invert: false
      action: route
      server: local
    - type: logical
      mode: and
      rules: []
      action: route
      server: local
```

!!! note ""

    当内容只有一项时，可以直接使用单个值，无需数组

## 默认字段

!!! note ""

    默认规则使用以下匹配逻辑:  
    (`domain` || `domain_suffix` || `domain_keyword` || `domain_regex` || `geosite` || `ip_cidr` || `ip_is_private` || `ip_accept_any`) &&  
    (`port` || `port_range`) &&  
    (`source_geoip` || `source_ip_cidr` || `source_ip_is_private`) &&  
    (`source_port` || `source_port_range`) &&  
    `其他字段`

    当规则集仅包含一条默认规则且非 invert 时，其中字段视为按以上规则与外层规则合并；否则，作为一条 `其他字段` 匹配；不同规则集之间始终保持 or。

### inbound

[入站](/zh/configuration/inbound/) 标签.

### ip_version

4 (A DNS 查询) 或 6 (AAAA DNS 查询)。

默认不限制。

### query_type

DNS 查询类型。值可以为整数或者类型名称字符串。

### query_client_subnet

匹配查询中的 `edns0-subnet` OPT 附加记录（EDNS 客户端子网）。

列出的前缀在不比收到的客户端子网更具体、且包含其地址时匹配。

如果值是 IP 地址而不是前缀，则会自动附加 `/32` 或 `/128`。

### query_dnssec

匹配设置了 DNSSEC OK (`DO`) 位的查询。

### network

`tcp` 或 `udp`。

### auth_user

认证用户名，参阅入站设置。

### protocol

探测到的协议, 参阅 [协议探测](/zh/configuration/route/sniff/)。

### domain

匹配完整域名。

### domain_suffix

匹配域名后缀。

### domain_keyword

匹配域名关键字。

### domain_regex

匹配域名正则表达式。

### geosite

匹配 Geosite。

### source_geoip

匹配源 GeoIP。

### source_ip_cidr

匹配源 IP CIDR。

### source_ip_is_private

匹配非公开源 IP。

### source_port

匹配源端口。

### source_port_range

匹配源端口范围。

### port

匹配端口。

### port_range

匹配端口范围。

### process_name

!!! quote ""

    仅支持 Linux、Windows 和 macOS.

匹配进程名称。

### process_path

!!! quote ""

    仅支持 Linux、Windows 和 macOS.

匹配进程路径。

### process_path_regex

!!! quote ""

    仅支持 Linux、Windows 和 macOS.

使用正则表达式匹配进程路径。

### package_name

匹配 Android 应用包名。

### package_name_regex

使用正则表达式匹配 Android 应用包名。

### user

!!! quote ""

    仅支持 Linux。

匹配用户名。

### user_id

!!! quote ""

    仅支持 Linux。

匹配用户 ID。

### clash_mode

匹配 Clash 模式。

### network_type

!!! quote ""

    仅在 Android 与 Apple 平台图形客户端中支持。

匹配网络类型。

Available values: `wifi`, `cellular`, `ethernet` and `other`.

### network_is_expensive

!!! quote ""

    仅在 Android 与 Apple 平台图形客户端中支持。

匹配如果网络被视为计费 (在 Android) 或被视为昂贵，
像蜂窝网络或个人热点 (在 Apple 平台)。

### network_is_constrained

!!! quote ""

    仅在 Apple 平台图形客户端中支持。

匹配如果网络在低数据模式下。

### interface_address

!!! quote ""

    仅支持 Linux、Windows 和 macOS.

匹配接口地址。

### network_interface_address

!!! quote ""

    仅在 Android 与 Apple 平台图形客户端中支持。

匹配网络接口（可用值同 `network_type`）地址。

### default_interface_address

!!! quote ""

    仅支持 Linux、Windows 和 macOS.

匹配默认接口地址。

### source_mac_address

!!! quote ""

    仅支持 Linux、macOS，或在 Android 和 macOS 图形客户端中支持。参阅 [邻居解析](/configuration/shared/neighbor/) 了解设置方法。

匹配源设备 MAC 地址。

### source_hostname

!!! quote ""

    仅支持 Linux、macOS，或在 Android 和 macOS 图形客户端中支持。参阅 [邻居解析](/configuration/shared/neighbor/) 了解设置方法。

匹配源设备从 DHCP 租约获取的主机名。

### preferred_by

匹配指定 DNS 服务器的首选域名。

| 类型            | 匹配                                                          |
|---------------|-------------------------------------------------------------|
| `hosts`       | 匹配预定义条目和 hosts 文件中的条目                                       |
| `local`       | 匹配 hosts 中的条目、邻居解析得到的主机名以及 mDNS 本地域名                         |
| `mdns`        | 匹配 mDNS 本地域名（`*.local.` 以及 IPv4/IPv6 链路本地反向区域）              |
| `tailscale`   | 匹配 MagicDNS 主机和 DNS 路由后缀                                    |
| `openconnect` | 匹配 VPN 服务器推送的分流 DNS 和搜索域                                  |
| `resolved`    | 匹配 systemd-resolved 链路中的分流域名和搜索域                            |

### dns_server_address

匹配指定 DNS 服务器的服务器地址。

| 类型            | 匹配                               |
|---------------|----------------------------------|
| `local`       | 匹配系统 DNS 服务器                     |
| `dhcp`        | 匹配通过 DHCP 获取的 DNS 服务器             |
| `resolved`    | 匹配 systemd-resolved 链路中的 DNS 服务器  |
| `tailscale`   | 匹配 tailnet 的 DNS 解析器              |
| `openvpn`     | 匹配 VPN 服务器推送的 DNS 服务器             |
| `openconnect` | 匹配 VPN 服务器推送的 DNS 服务器             |

### dns_search_domain

匹配指定 DNS 服务器的搜索域。

| 类型            | 匹配                            |
|---------------|-------------------------------|
| `local`       | 匹配系统搜索域                       |
| `dhcp`        | 匹配通过 DHCP 获取的搜索域              |
| `resolved`    | 匹配 systemd-resolved 链路中的搜索域    |
| `tailscale`   | 匹配 tailnet 的搜索域                |
| `openvpn`     | 匹配 VPN 服务器推送的搜索域              |
| `openconnect` | 匹配 VPN 服务器推送的搜索域              |

### wifi_ssid

!!! quote ""

    仅在 Android 与 Apple 平台图形客户端和 Linux 中支持。

匹配 WiFi SSID。

### wifi_bssid

!!! quote ""

    仅在 Android 与 Apple 平台图形客户端和 Linux 中支持。

匹配 WiFi BSSID。

### rule_set

匹配[规则集](/zh/configuration/route/#rule_set)。

### rule_set_ip_cidr_match_source

使规则集中的 `ip_cidr` 规则匹配源 IP。

### match_response

启用响应匹配。启用后，此规则将匹配已评估的响应（由前序 [`evaluate`](/zh/configuration/dns/rule_action/#evaluate) 动作设置），而不仅是匹配原始查询。

可以为 `true` 或 `evaluate` 动作的 `tag`：`true` 匹配最近一条无 `tag` 的 `evaluate` 动作的响应；标签则匹配对应 `evaluate` 动作的响应。

该已评估的响应也可以被后续的 [`respond`](/zh/configuration/dns/rule_action/#respond) 动作直接返回；在带 `match_response` 标签的规则中，`respond` 返回该标签的响应。

响应匹配字段（`response_rcode`、`response_answer`、`response_ns`、`response_extra`）需要此选项。
当与 `evaluate` 或响应匹配字段一起使用时，`ip_cidr`、`ip_is_private` 和 `ip_accept_any` 也需要此选项。

### ip_accept_any

当 DNS 查询响应包含至少一个地址时匹配。

### invert

反选匹配结果。

### action

**必填。**参阅 [规则动作](/zh/configuration/dns/rule_action/)。

### server

### disable_cache

### rewrite_ttl

### client_subnet

## 旧版地址筛选字段

仅对地址请求 (A/AAAA/HTTPS) 生效。 当查询结果与地址筛选规则项不匹配时，将跳过当前规则。

!!! info ""

    引用的规则集中的 `ip_cidr` 项也作为地址筛选字段生效。

### ip_cidr

与查询响应匹配 IP CIDR。

作为旧版地址筛选字段已废弃。请改为配合 `match_response` 使用，
参阅[迁移指南](/zh/migration/#迁移地址筛选字段到响应匹配)。

### ip_is_private

与查询响应匹配非公开 IP。

作为旧版地址筛选字段已废弃。请改为配合 `match_response` 使用，
参阅[迁移指南](/zh/migration/#迁移地址筛选字段到响应匹配)。

## 响应匹配字段

已评估的响应的匹配字段。需要将 `match_response` 设为 `true`，
且需要前序规则使用 [`evaluate`](/zh/configuration/dns/rule_action/#evaluate) 动作来填充响应。

该已评估的响应也可以被后续的 [`respond`](/zh/configuration/dns/rule_action/#respond) 动作直接返回。

### response_rcode

匹配 DNS 响应码。

接受的值与 [predefined 动作 rcode](/zh/configuration/dns/rule_action/#rcode) 中相同。

### response_answer

匹配 DNS 应答记录。

记录格式与 [predefined 动作 answer](/zh/configuration/dns/rule_action/#answer) 中相同。

### response_ns

匹配 DNS 名称服务器记录。

记录格式与 [predefined 动作 ns](/zh/configuration/dns/rule_action/#ns) 中相同。

### response_extra

匹配 DNS 额外记录。

记录格式与 [predefined 动作 extra](/zh/configuration/dns/rule_action/#extra) 中相同。

## 逻辑字段

### type

`logical`

### mode

**必填。**`and` 或 `or`

### rules

**必填。**包括的规则。
