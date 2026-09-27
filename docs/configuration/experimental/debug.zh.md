# 调试

```{.yaml linenums="1"}
listen: ""
gc_percent: 0
max_stack: 0
max_threads: 0
panic_on_fault: false
trace_back: ""
memory_limit: 0
```

!!! note ""

    `debug` 必须是对象。旧版布尔形式的 `debug` 会被拒绝。

## listen

调试 HTTP 服务器的监听地址。

服务器提供 `/debug/gc`、`/debug/memory` 和 `/debug/pprof` 端点。

如果为空则禁用。

## gc_percent

通过 `runtime/debug.SetGCPercent` 设置垃圾回收目标百分比。

如果为空则使用 Go 运行时默认值。

## max_stack

通过 `runtime/debug.SetMaxStack` 设置每个 goroutine 的最大栈大小（字节）。

如果为空则使用 Go 运行时默认值。

## max_threads

通过 `runtime/debug.SetMaxThreads` 设置操作系统线程的最大数量。

如果为空则使用 Go 运行时默认值。

## panic_on_fault

通过 `runtime/debug.SetPanicOnFault` 设置当发生程序错误（例如无效的内存地址）时是否触发 panic 而不是使进程崩溃。

默认禁用。

## trace_back

通过 `runtime/debug.SetTraceback` 设置 traceback 级别。

可选值为 `none`、`single`、`all`、`system` 或 `crash`。

如果为空则使用 Go 运行时默认值。

## memory_limit

通过 `runtime/debug.SetMemoryLimit` 设置 Go 运行时软内存限制。

配置的值在应用前会除以 `1.5`，因此实际运行时限制低于配置值。

如果为空或为 `0` 则不设置限制。

## oom_killer

已废弃。用于启用 OOM 处理的旧版布尔字段。

该字段已被移除并在加载时被拒绝；请改用 [OOM 终结者](/zh/configuration/service/oom-killer/) 服务。
