# 缓存文件

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

启用缓存文件。

## path

缓存文件路径，默认使用`cache.db`。

## cache_id

缓存文件中的标识符。

如果不为空，配置特定的数据将使用由其键控的单独存储。

## store_fakeip

将 fakeip 存储在缓存文件中。

## store_dns

将 DNS 缓存存储在缓存文件中。

## buffer_size

写缓存的大小。

默认使用 `1MB`。

## flush_interval

自动冲刷写缓存的间隔。

默认禁用。
