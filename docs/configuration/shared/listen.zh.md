# 监听字段

```{.yaml linenums="1"}
listen: ""
listen_port: 0
bind_interface: ""
routing_mark: 0
reuse_addr: false
netns: ""
tcp_fast_open: false
tcp_multi_path: false
disable_tcp_keep_alive: false
tcp_keep_alive: ""
tcp_keep_alive_interval: ""
udp_fragment: false
udp_timeout: ""
detour: ""
```

## listen

**必填。**监听地址。

## listen_port

监听端口。

## bind_interface

要绑定到的网络接口。

## routing_mark

!!! quote ""

    仅支持 Linux。

设置 netfilter 路由标记。

支持数字 (如 `1234`) 和十六进制字符串 (如 `"0x1234"`)。

## reuse_addr

重用监听地址。

## netns

!!! quote ""

    仅支持 Linux。

设置网络命名空间，名称或路径。

自 sing-box 1.14.0 起，也可以使用[网络命名空间](/zh/configuration/network-namespace/)的标签。

## tcp_fast_open

启用 TCP Fast Open。

## tcp_multi_path

!!! warning ""

    需要 Go 1.21。

启用 TCP Multi Path。

## disable_tcp_keep_alive

禁用 TCP keep alive。

## tcp_keep_alive

TCP keep alive 初始周期。

默认使用 `5m`。

## tcp_keep_alive_interval

TCP keep alive 间隔。

默认使用 `75s`。

## udp_fragment

启用 UDP 分段。

## udp_timeout

UDP NAT 过期时间。

默认使用 `5m`。

## detour

如果设置，连接将被转发到指定的入站。

需要目标入站支持，参阅 [注入支持](/zh/configuration/inbound/)。
