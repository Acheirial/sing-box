# TCP Brutal

## Server Requirements

* Linux
* `brutal` congestion control algorithm kernel module installed

See [tcp-brutal](https://github.com/apernet/tcp-brutal) for details.

```{.yaml linenums="1"}
enabled: true
up_mbps: 100
down_mbps: 100
```

## enabled

Enable TCP Brutal congestion control algorithm。

## up_mbps, down_mbps

**Required.** Upload and download bandwidth, in Mbps.
