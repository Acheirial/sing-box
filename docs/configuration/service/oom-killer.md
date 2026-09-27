# OOM Killer

OOM Killer service monitors memory usage and releases memory before the process runs out of memory.

When memory pressure is detected, the service resets the network, flushes the cache file and forces the Go runtime to release memory to the operating system.

See also: [Debug](/configuration/experimental/debug/)

```{.yaml linenums="1"}
type: oom-killer

memory_limit: 0
safety_margin: 0
min_interval: ""
max_interval: ""
```

!!! note ""

    This `memory_limit` is independent of `experimental.debug.memory_limit`: the latter configures the Go runtime soft memory limit, while this one drives the service's own memory pressure detection.

## memory_limit

Memory limit enforced by the service.

Memory pressure is triggered when memory usage approaches this limit.

If empty, the service monitors available system memory instead on platforms where it is available.

## safety_margin

Amount of memory kept below `memory_limit` before memory pressure is triggered.

`5MiB` is used by default when `memory_limit` is set.

## min_interval

Minimum interval between memory polls.

`100ms` is used by default.

Must be greater than 0.

## max_interval

Maximum interval between memory polls.

`10s` is used by default.

Must be greater than or equal to `min_interval`.
