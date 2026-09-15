---
icon: material/new-box
---

!!! question "自 sing-box 1.15.0 起"

# YAML

sing-box 除默认的 JSON 格式外，还接受扩展名为 `.yaml` 或 `.yml` 的 YAML 配置文件。

### 工作原理

YAML 配置文件会被解析并在内部转换为 JSON，然后与其他配置一样处理。

所有配置选项与 JSON 格式完全相同。文件扩展名决定解析器：以 `.yaml` 或 `.yml` 结尾的文件被视为 YAML，其余文件被视为 JSON。

### 示例

以下 YAML 配置等价于一个基本的 JSON 配置：

```yaml
log:
  level: info
inbounds:
  - type: mixed
    tag: mixed-in
    listen: 127.0.0.1
    listen_port: 2080
outbounds:
  - type: direct
    tag: direct-out
```

### 限制

* 从标准输入读取的配置（`-c stdin`）必须为 JSON 格式。
* 通过库 API（libbox/daemon）提供的配置仍仅支持 JSON。
* `sing-box format -w` 会将格式化后的 JSON 写回文件。由于 YAML 在加载时会被转换，格式化后磁盘上的文件将变为 JSON。
