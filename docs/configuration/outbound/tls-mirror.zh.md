---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# TLS Mirror

### 结构

```yaml
type: tls-mirror
tag: tls-mirror-out
server: 127.0.0.1
server_port: 443
tls: {}
primary_key: ""
explicit_nonce_ciphersuites: []
defer_instance_derived_write_time:
  base_nanoseconds: 0
  uniform_random_multiplier_nanoseconds: 0
transport_layer_padding:
  enabled: false
connection_enrolment:
  primary_ingress_outbound: ""
  primary_egress_outbound: ""
embedded_traffic_generator:
  steps:
    - name: ""
      host: ""
      path: ""
      method: ""
      headers:
        - name: ""
          value: ""
          values: []
      next_step:
        - weight: 0
          goto_location: 0
      connection_ready: false
      connection_recall_exit: false
      wait_time:
        base_nanoseconds: 0
        uniform_random_multiplier_nanoseconds: 0
      h2_do_not_wait_for_download_finish: false
sequence_watermarking_enabled: false
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#outbound)。

#### primary_key

==必填==

用于派生镜像状态的主密钥。

#### primary_ingress_outbound

用于连接注册的主入站连接的出站标签。位于 `connection_enrolment` 内；启用注册时，`primary_ingress_outbound` 与 `primary_egress_outbound` 至少设置其一，且引用的出站不能是 TLS Mirror 出站本身。

#### primary_egress_outbound

用于连接注册的主出站连接的出站标签。为空时回退到 `primary_ingress_outbound`。

#### explicit_nonce_ciphersuites

携带显式 nonce 的 TLS 1.2 密码套件 ID 列表（整数）。

#### defer_instance_derived_write_time

延迟实例派生写入的时间规格。

| 字段                                     | 说明                          |
|:----------------------------------------|:------------------------------|
| `base_nanoseconds`                      | 基础延迟（纳秒）               |
| `uniform_random_multiplier_nanoseconds` | 在基础延迟之上叠加的均匀随机乘数（纳秒） |

#### transport_layer_padding

传输层填充选项。

| 字段      | 说明            |
|:----------|:----------------|
| `enabled` | 启用传输层填充    |

#### connection_enrolment

连接注册选项，参阅 `primary_ingress_outbound` 与 `primary_egress_outbound`。

#### embedded_traffic_generator

用于塑造连接流量外观的内嵌流量生成器。

每个 step 支持：

| 字段                                    | 说明                                                    |
|:----------------------------------------|:--------------------------------------------------------|
| `name`                                  | step 名称                                                |
| `host`                                  | 请求 Host                                                |
| `path`                                  | 请求路径                                                  |
| `method`                                | 请求方法                                                  |
| `headers`                               | 请求头，每项包含 `name` 与 `value` 或 `values`            |
| `next_step`                             | 转移候选，每项包含 `weight` 与 `goto_location`             |
| `connection_ready`                      | 将连接标记为就绪                                          |
| `connection_recall_exit`                | 召回连接后退出                                             |
| `wait_time`                             | 等待时间规格，同 `defer_instance_derived_write_time`       |
| `h2_do_not_wait_for_download_finish`    | 不等待 HTTP/2 下载完成                                     |

#### sequence_watermarking_enabled

启用序列号水印。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/)。
