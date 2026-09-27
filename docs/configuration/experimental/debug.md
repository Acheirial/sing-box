# Debug

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

    `debug` must be an object. The legacy boolean form of `debug` is rejected.

## listen

Address for the debug HTTP server.

The server exposes `/debug/gc`, `/debug/memory` and `/debug/pprof` endpoints.

Disabled if empty.

## gc_percent

Sets the garbage collection target percentage via `runtime/debug.SetGCPercent`.

The Go runtime default is used if empty.

## max_stack

Sets the maximum stack size per goroutine in bytes via `runtime/debug.SetMaxStack`.

The Go runtime default is used if empty.

## max_threads

Sets the maximum number of operating system threads via `runtime/debug.SetMaxThreads`.

The Go runtime default is used if empty.

## panic_on_fault

Sets whether a program fault, such as an invalid memory address, panics instead of crashing the process, via `runtime/debug.SetPanicOnFault`.

Disabled by default.

## trace_back

Sets the traceback level via `runtime/debug.SetTraceback`.

One of `none`, `single`, `all`, `system` or `crash`.

The Go runtime default is used if empty.

## memory_limit

Sets the Go runtime soft memory limit via `runtime/debug.SetMemoryLimit`.

The configured value is divided by `1.5` before being applied, so the effective runtime limit is lower than the configured value.

No limit is set if empty or `0`.

## oom_killer

Deprecated. A legacy boolean field previously used to enable OOM handling.

It is removed and rejected on load; use the [OOM Killer](/configuration/service/oom-killer/) service instead.
