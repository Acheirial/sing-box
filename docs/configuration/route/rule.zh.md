# 路由规则

```{.yaml linenums="1"}
route:
  rules:
    - inbound:
        - mixed-in
      ip_version: 6
      network:
        - tcp
      auth_user:
        - usera
        - userb
      protocol:
        - tls
        - http
        - quic
      client:
        - chromium
        - safari
        - firefox
        - quic-go
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
      source_ip_is_private: false
      ip_cidr:
        - 10.0.0.0/24
      ip_is_private: false
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
      wifi_ssid:
        - My WIFI
      wifi_bssid:
        - "00:00:00:00:00:00"
      preferred_by:
        - tailscale
        - wireguard
      dns_server_address:
        local:
          - 192.168.1.1/32
      dns_search_domain:
        ts-dns:
          - example.ts.net
      source_mac_address:
        - "00:11:22:33:44:55"
      source_hostname:
        - my-device
      rule_set:
        - geoip-cn
        - geosite-cn
      rule_set_ip_cidr_match_source: false
      invert: false
      action: route
      outbound: direct
    - type: logical
      mode: and
      rules: []
      invert: false
      action: route
      outbound: direct
```

!!! note ""

    当内容只有一项时，可以直接使用单个值，无需数组。

## 默认字段

!!! note ""

    默认规则使用以下匹配逻辑:  
    (`domain` || `domain_suffix` || `domain_keyword` || `domain_regex` || `ip_cidr` || `ip_is_private`) &&  
    (`port` || `port_range`) &&  
    (`source_ip_cidr` || `source_ip_is_private`) &&  
    (`source_port` || `source_port_range`) &&  
    `其他字段`

    当规则集仅包含一条默认规则且非 invert 时，其中字段视为按以上规则与外层规则合并；否则，作为一条 `其他字段` 匹配；不同规则集之间始终保持 or。

### inbound

[入站](/zh/configuration/inbound/) 标签。

### ip_version

4 或 6。

默认不限制。

### auth_user

认证用户名，参阅入站设置。

### protocol

探测到的协议, 参阅 [协议探测](/zh/configuration/route/sniff/)。

### client

探测到的客户端类型, 参阅 [协议探测](/zh/configuration/route/sniff/)。

### network

匹配网络类型。

`tcp`、`udp` 或 `icmp`。

### domain

匹配完整域名。

### domain_suffix

匹配域名后缀。

### domain_keyword

匹配域名关键字。

### domain_regex

匹配域名正则表达式。

### source_ip_cidr

匹配源 IP CIDR。

### source_ip_is_private

匹配非公开源 IP。

### ip_cidr

匹配 IP CIDR。

### ip_is_private

匹配非公开 IP。

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

    仅支持 Linux、Windows 和 macOS。

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

    仅支持 Linux.

匹配用户名。

### user_id

!!! quote ""

    仅支持 Linux.

匹配用户 ID。

### clash_mode

匹配 Clash 模式。

### network_type

!!! quote ""

    仅在 Android 与 Apple 平台图形客户端中支持。

匹配网络类型。

可用值: `wifi`, `cellular`, `ethernet` and `other`.

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

### wifi_ssid

匹配 WiFi SSID。

参阅 [Wi-Fi 状态](/zh/configuration/shared/wifi-state/)。

### wifi_bssid

匹配 WiFi BSSID。

参阅 [Wi-Fi 状态](/zh/configuration/shared/wifi-state/)。

### preferred_by

匹配制定出站的首选路由。

| 类型          | 匹配                             |
|-------------|--------------------------------|
| `tailscale` | 匹配 MagicDNS 域名和对端的 allowed IPs |
| `wireguard` | 匹配对端的 allowed IPs              |
| `bridge`    | 匹配除本机本地地址外的所有地址，仅在[预匹配](/zh/configuration/shared/pre-match/)中 |

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

### source_mac_address

!!! quote ""

    仅支持 Linux、macOS，或在 Android 和 macOS 图形客户端中支持。参阅 [邻居解析](/configuration/shared/neighbor/) 了解设置方法。

匹配源设备 MAC 地址。

### source_hostname

!!! quote ""

    仅支持 Linux、macOS，或在 Android 和 macOS 图形客户端中支持。参阅 [邻居解析](/configuration/shared/neighbor/) 了解设置方法。

匹配源设备从 DHCP 租约获取的主机名。

### rule_set

匹配[规则集](/zh/configuration/route/#rule_set)。

### rule_set_ip_cidr_match_source

使规则集中的 `ip_cidr` 规则匹配源 IP。

### invert

反选匹配结果。

### action

**必填。**参阅 [规则动作](/zh/configuration/route/rule_action/)。

### outbound

## 逻辑字段

### type

`logical`

### mode

**必填。** `and` 或 `or`

### rules

**必填。**包括的规则。
