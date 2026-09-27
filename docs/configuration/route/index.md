# Route

```{.yaml linenums="1"}
route:
  rules: []
  rule_set: []
  final: ""
  auto_detect_interface: false
  override_android_vpn: false
  default_interface: ""
  default_mark: 0
  find_process: false
  find_neighbor: false
  dhcp_lease_files: []
  default_http_client: ""
  default_domain_resolver: ""  # or {}
  default_network_strategy: ""
  default_network_type: []
  default_fallback_network_type: []
  default_fallback_delay: ""

  # Removed

  geoip: {}
  geosite: {}
```

!!! note ""

    You can use a single value instead of an array when the content is only one item

## rules

List of [Route Rule](./rule/)

## rule_set

List of [rule-set](/configuration/rule-set/)

## final

Default outbound tag. the first outbound will be used if empty.

## auto_detect_interface

!!! quote ""

    Only supported on Linux, Windows and macOS.

Bind outbound connections to the default NIC by default to prevent routing loops under tun.

Takes no effect if `outbound.bind_interface` is set.

## override_android_vpn

!!! quote ""

    Only supported on Android.

Accept Android VPN as upstream NIC when `auto_detect_interface` enabled.

## default_interface

!!! quote ""

    Only supported on Linux, Windows and macOS.

Bind outbound connections to the specified NIC by default to prevent routing loops under tun.

Takes no effect if `auto_detect_interface` is set.

## default_mark

!!! quote ""

    Only supported on Linux.

Set routing mark by default.

Takes no effect if `outbound.routing_mark` is set.

## find_process

!!! quote ""

    Only supported on Linux, Windows, and macOS.

Enable process search for logging when no `process_name`, `process_path`, `package_name`, `user` or `user_id` rules exist.

## find_neighbor

!!! quote ""

    Only supported on Linux and macOS.

Enable neighbor resolution for logging when no `source_mac_address` or `source_hostname` rules exist.

See [Neighbor Resolution](/configuration/shared/neighbor/) for setup.

## dhcp_lease_files

!!! quote ""

    Only supported on Linux and macOS.

Custom DHCP lease file paths for hostname and MAC address resolution.

Automatically detected from common DHCP servers (dnsmasq, odhcpd, ISC dhcpd, Kea) if empty.

## default_http_client

Tag of the default [HTTP Client](/configuration/shared/http-client/) used by remote rule-sets.

If empty and `http_clients` is defined, the first HTTP client is used.

## default_domain_resolver

See [Dial Fields](/configuration/shared/dial/#domain_resolver) for details.

Can be overridden by `outbound.domain_resolver`.

## default_network_strategy

See [Dial Fields](/configuration/shared/dial/#network_strategy) for details.

Takes no effect if `outbound.bind_interface`, `outbound.inet4_bind_address` or `outbound.inet6_bind_address` is set.

Can be overridden by `outbound.network_strategy`.

Conflicts with `default_interface`.

## default_network_type

See [Dial Fields](/configuration/shared/dial/#network_type) for details.

## default_fallback_network_type

See [Dial Fields](/configuration/shared/dial/#fallback_network_type) for details.

## default_fallback_delay

See [Dial Fields](/configuration/shared/dial/#fallback_delay) for details.
