# OOM Killer

OOM 终结者服务监控内存使用情况，并在进程耗尽内存之前释放内存。

当检测到内存压力时，服务会重置网络、刷新缓存文件并强制 Go 运行时将内存释放回操作系统。

另请参阅：[调试](/zh/configuration/experimental/debug/)

```{.yaml linenums="1"}
type: oom-killer

memory_limit: 0
safety_margin: 0
min_interval: ""
max_interval: ""
```

!!! note ""

    此处的 `memory_limit` 与 `experimental.debug.memory_limit` 相互独立：后者配置 Go 运行时软内存限制，而前者用于驱动服务自身的内存压力检测。

## memory_limit

服务强制执行的内存限制。

当内存使用量接近此限制时会触发内存压力。

如果为空，服务会在平台支持的情况下改为监控系统可用内存。

## safety_margin

在触发内存压力之前，在 `memory_limit` 之下保留的内存量。

当设置了 `memory_limit` 时，默认使用 `5MiB`。

## min_interval

内存轮询之间的最小间隔。

默认使用 `100ms`。

必须大于 0。

## max_interval

内存轮询之间的最大间隔。

默认使用 `10s`。

必须大于或等于 `min_interval`。
