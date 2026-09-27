# Cache File

```{.yaml linenums="1"}
enabled: true
path: ""
cache_id: ""
store_fakeip: false
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

## store_dns

Store DNS cache in the cache file.

## buffer_size

Size of the write buffer.

`1MB` is used by default.

## flush_interval

Interval for flushing the write buffer automatically.

Disabled by default.
