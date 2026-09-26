# 引言

sing-box 配置文件使用 YAML 格式编写。
`-c/--config` 与 `-C/--config-directory` 选项会自动识别 `.yaml`、`.yml` 与 `.json` 扩展名。
对于 `stdin` 与未识别的扩展名，将根据内容特征自动检测格式。

支持 YAML 锚点（`&`）、别名（`*`）以及合并键（`<<`）。

### 结构

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
sing-box format -w -c config.yaml -D config_directory
```

输出将保持 YAML 格式，键按字母排序，注释不保留。

### 合并

```bash
sing-box merge output.yaml -c config.yaml -D config_directory
```

### JSON 支持

同样支持使用 JSON 格式的配置文件，它与 YAML 配置文件共享完全相同的配置结构与校验规则。

规则集（rule-set）文件及其他独立外部资源仍然使用 JSON 或对应的二进制格式。
