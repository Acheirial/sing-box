# 引言

sing-box 支持使用 JSON 与 YAML 作为配置文件格式。
`-c/--config` 与 `-C/--config-directory` 选项会自动识别 `.json`、`.yaml` 与 `.yml` 扩展名。
对于 `stdin` 与未识别的扩展名，将根据内容特征自动检测格式。
### 结构

```json
{
  "$schema": "https://sing-box.sagernet.org/schema.json",
  "log": {},
  "dns": {},
  "ntp": {},
  "certificate": {},
  "certificate_providers": [],
  "http_clients": [],
  "network_namespaces": [],
  "endpoints": [],
  "inbounds": [],
  "outbounds": [],
  "route": {},
  "services": [],
  "experimental": {}
}
```

等价的 YAML 配置：

```yaml
$schema: https://sing-box.sagernet.org/schema.json
log: {}
dns: {}
ntp: {}
certificate: {}
certificate_providers: []
http_clients: []
network_namespaces: []
endpoints: []
inbounds: []
outbounds: []
route: {}
services: []
experimental: {}
```

### 字段

| Key            | Format                 |
|----------------|------------------------|
| `$schema`      | [JSON Schema](./schema/) |
| `log`          | [日志](./log/)           |
| `dns`          | [DNS](./dns/)          |
| `ntp`          | [NTP](./ntp/)          |
| `certificate`  | [证书](./certificate/)   |
| `certificate_providers` | [证书提供者](./shared/certificate-provider/) |
| `http_clients` | [HTTP 客户端](./shared/http-client/) |
| `network_namespaces` | [网络命名空间](./network-namespace/) |
| `endpoints`    | [端点](./endpoint/)      |
| `inbounds`     | [入站](./inbound/)       |
| `outbounds`    | [出站](./outbound/)      |
| `route`        | [路由](./route/)         |
| `services`     | [服务](./service/)       |
| `experimental` | [实验性](./experimental/) |

### 检查

```bash
sing-box check
```

### 格式化

```bash
sing-box format -w -c config.json -D config_directory
```

### 合并

```bash
sing-box merge output.json -c config.json -D config_directory
```

### YAML 支持

YAML 配置文件与 JSON 配置文件共享完全相同的配置结构与校验规则。
支持 YAML 锚点（`&`）、别名（`*`）以及合并键（`<<`）。

对 YAML 文件使用 `sing-box format` 时，输出将保持 YAML 格式（键按字母排序，注释不保留）。
`sing-box merge` 输出路径以 `.yaml` 或 `.yml` 结尾时，合并结果将编码为 YAML。

规则集（rule-set）文件及其他独立外部资源仍然使用 JSON 或对应的二进制格式。
