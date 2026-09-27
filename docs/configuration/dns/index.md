# DNS

```{.yaml linenums="1"}
dns:
  servers: []
  rules: []
  final: ""
  strategy: ""
  disable_cache: false
  disable_expire: false
  cache_capacity: 0
  optimistic: false  # or {}
  timeout: ""
  reverse_mapping: false
  client_subnet: ""
```

| Key      | Format                          |
|----------|---------------------------------|
| `server` | List of [DNS Server](./server/) |
| `rules`  | List of [DNS Rule](./rule/)     |

## final

Default dns server tag.

The first server will be used if empty.

## strategy

Default domain strategy for resolving the domain names.

One of `prefer_ipv4` `prefer_ipv6` `ipv4_only` `ipv6_only`.

## disable_cache

Disable dns cache.

Conflict with `optimistic`.

## disable_expire

Disable dns cache expire.

Conflict with `optimistic`.

## cache_capacity

LRU cache capacity.

Value less than 1024 will be ignored.

## optimistic

Enable optimistic DNS caching. When a cached DNS entry has expired but is still within the timeout window,
the stale response is returned immediately while a background refresh is triggered.

Conflict with `disable_cache` and `disable_expire`.

Accepts a boolean or an object. When set to `true`, the default timeout of `3d` is used.

```{.yaml linenums="1"}
enabled: true
timeout: 3d
```

### enabled

Enable optimistic DNS caching.

### timeout

The maximum time an expired cache entry can be served optimistically.

`3d` is used by default.

## timeout

Default timeout for each DNS query.

`10s` is used by default.

Can be overridden by `rules.[].timeout` (DNS rule action) or `domain_resolver.timeout`.

## reverse_mapping

Stores a reverse mapping of IP addresses after responding to a DNS query in order to provide domain names when routing.

Since this process relies on the act of resolving domain names by an application before making a request, it can be
problematic in environments such as macOS, where DNS is proxied and cached by the system.

## client_subnet

Append a `edns0-subnet` OPT extra record with the specified IP prefix to every query by default.

If value is an IP address instead of prefix, `/32` or `/128` will be appended automatically.

Can be overridden by `servers.[].client_subnet` or `rules.[].client_subnet`.
