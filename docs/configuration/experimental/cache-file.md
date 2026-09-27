# Cache File

!!! question "Since sing-box 1.8.0"

!!! quote "Changes in sing-box 1.14.0"

    :material-delete-clock: [store_rdrc](#store_rdrc)  
    :material-delete-clock: [rdrc_timeout](#rdrc_timeout)  
    :material-plus: [store_dns](#store_dns)

!!! quote "Changes in sing-box 1.9.0"

    :material-plus: [store_rdrc](#store_rdrc)  
    :material-plus: [rdrc_timeout](#rdrc_timeout)

```{.yaml linenums="1"}
enabled: true
path: ""
cache_id: ""
store_fakeip: false
store_rdrc: false
rdrc_timeout: ""
store_dns: false
buffer_size: ""
flush_interval: ""
```

## enabled

Enable cache file.

## path

Path to the cache file.

`cache.db` will be used if empty.

## cache_id

Identifier in the cache file

If not empty, configuration specified data will use a separate store keyed by it.

## store_fakeip

Store fakeip in the cache file

## store_rdrc

!!! failure "Deprecated in sing-box 1.14.0"

    `store_rdrc` is deprecated and will be removed in sing-box 1.16.0, check [Migration](/migration/#migrate-store-rdrc).

Store rejected DNS response cache in the cache file

The check results of [Legacy Address Filter Fields](/configuration/dns/rule/#legacy-address-filter-fields)
will be cached until expiration.

## rdrc_timeout

!!! failure "Deprecated in sing-box 1.14.0"

    `rdrc_timeout` is deprecated and will be removed in sing-box 1.16.0, check [Migration](/migration/#migrate-store-rdrc).

Timeout of rejected DNS response cache.

`7d` is used by default.

## store_dns

!!! question "Since sing-box 1.14.0"

Store DNS cache in the cache file.

## buffer_size

Size of the write buffer.

`1MB` is used by default.

## flush_interval

Interval for flushing the write buffer automatically.

Disabled by default.
