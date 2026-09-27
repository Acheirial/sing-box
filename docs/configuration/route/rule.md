# Route Rule

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
        - 192.168.0.1
      source_ip_is_private: false
      ip_cidr:
        - 10.0.0.0/24
        - 192.168.0.1
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

    You can use a single value instead of an array when the content is only one item

## Default Fields

!!! note ""

    The default rule uses the following matching logic:  
    (`domain` || `domain_suffix` || `domain_keyword` || `domain_regex` || `ip_cidr` || `ip_is_private`) &&  
    (`port` || `port_range`) &&  
    (`source_ip_cidr` || `source_ip_is_private`) &&  
    (`source_port` || `source_port_range`) &&  
    `other fields`

    When a rule-set contains only a single default rule without `invert`, its fields are considered merged into the outer rule per the logic above; otherwise, it is matched as an `other field`; different rule-sets always keep OR semantics.

### inbound

Tags of [Inbound](/configuration/inbound/).

### ip_version

4 or 6.

Not limited if empty.

### auth_user

Username, see each inbound for details.

### protocol

Sniffed protocol, see [Protocol Sniff](/configuration/route/sniff/) for details.

### client

Sniffed client type, see [Protocol Sniff](/configuration/route/sniff/) for details.

### network

Match network type.

`tcp`, `udp` or `icmp`.

### domain

Match full domain.

### domain_suffix

Match domain suffix.

### domain_keyword

Match domain using keyword.

### domain_regex

Match domain using regular expression.

### source_ip_cidr

Match source IP CIDR.

### ip_is_private

Match non-public IP.

### ip_cidr

Match IP CIDR.

### source_ip_is_private

Match non-public source IP.

### source_port

Match source port.

### source_port_range

Match source port range.

### port

Match port.

### port_range

Match port range.

### process_name

!!! quote ""

    Only supported on Linux, Windows, and macOS.

Match process name.

### process_path

!!! quote ""

    Only supported on Linux, Windows, and macOS.

Match process path.

### process_path_regex

!!! quote ""

    Only supported on Linux, Windows, and macOS.

Match process path using regular expression.

### package_name

Match android package name.

### package_name_regex

Match android package name using regular expression.

### user

!!! quote ""

    Only supported on Linux.

Match user name.

### user_id

!!! quote ""

    Only supported on Linux.

Match user id.

### clash_mode

Match Clash mode.

### network_type

!!! quote ""

    Only supported in graphical clients on Android and Apple platforms.

Match network type.

Available values: `wifi`, `cellular`, `ethernet` and `other`.

### network_is_expensive

!!! quote ""

    Only supported in graphical clients on Android and Apple platforms.

Match if network is considered Metered (on Android) or considered expensive,
such as Cellular or a Personal Hotspot (on Apple platforms).

### network_is_constrained

!!! quote ""

    Only supported in graphical clients on Apple platforms.

Match if network is in Low Data Mode.

### interface_address

!!! quote ""

    Only supported on Linux, Windows, and macOS.

Match interface address.

### network_interface_address

!!! quote ""

    Only supported in graphical clients on Android and Apple platforms.

Matches network interface (same values as `network_type`) address.

### default_interface_address

!!! quote ""

    Only supported on Linux, Windows, and macOS.

Match default interface address.

### wifi_ssid

Match WiFi SSID.

See [Wi-Fi State](/configuration/shared/wifi-state/) for details.

### wifi_bssid

Match WiFi BSSID.

See [Wi-Fi State](/configuration/shared/wifi-state/) for details.

### preferred_by

Match specified outbounds' preferred routes.

| Type        | Match                                              |
|-------------|----------------------------------------------------|
| `tailscale` | Match MagicDNS domains and peers' allowed IPs      |
| `wireguard` | Match peers's allowed IPs                          |
| `bridge`    | Match all addresses except local addresses of the machine, only in [pre-match](/configuration/shared/pre-match/) |

### dns_server_address

Match specified DNS servers' server addresses.

| Type          | Match                                         |
|---------------|-----------------------------------------------|
| `local`       | Match system DNS servers                      |
| `dhcp`        | Match DNS servers from DHCP                   |
| `resolved`    | Match DNS servers from systemd-resolved links |
| `tailscale`   | Match DNS resolvers of the tailnet            |
| `openvpn`     | Match DNS servers pushed by the VPN server    |
| `openconnect` | Match DNS servers pushed by the VPN server    |

### dns_search_domain

Match specified DNS servers' search domains.

| Type          | Match                                            |
|---------------|--------------------------------------------------|
| `local`       | Match system search domains                      |
| `dhcp`        | Match search domains from DHCP                   |
| `resolved`    | Match search domains from systemd-resolved links |
| `tailscale`   | Match search domains of the tailnet              |
| `openvpn`     | Match search domains pushed by the VPN server    |
| `openconnect` | Match search domains pushed by the VPN server    |

### source_mac_address

!!! quote ""

    Only supported on Linux, macOS, or in graphical clients on Android and macOS. See [Neighbor Resolution](/configuration/shared/neighbor/) for setup.

Match source device MAC address.

### source_hostname

!!! quote ""

    Only supported on Linux, macOS, or in graphical clients on Android and macOS. See [Neighbor Resolution](/configuration/shared/neighbor/) for setup.

Match source device hostname from DHCP leases.

### rule_set

Match [rule-set](/configuration/route/#rule_set).

### rule_set_ip_cidr_match_source

Make `ip_cidr` in rule-sets match the source IP.

### invert

Invert match result.

### action

**Required.** See [Rule Actions](../rule_action/) for details.

### outbound

## Logical Fields

### type

`logical`

### mode

**Required.** `and` or `or`

### rules

**Required.** Included rules.
