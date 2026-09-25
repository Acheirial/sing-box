# Introduction

sing-box supports both JSON and YAML for configuration files.
Files ending in `.json`, `.yaml`, and `.yml` are automatically recognized
by the `-c/--config` and `-C/--config-directory` options.
For `stdin` and unrecognized extensions, format is detected from the content.
### Structure

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

Equivalent YAML configuration:

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

### Fields

| Key            | Format                          |
|----------------|---------------------------------|
| `$schema`      | [JSON Schema](./schema/)        |
| `log`          | [Log](./log/)                   |
| `dns`          | [DNS](./dns/)                   |
| `ntp`          | [NTP](./ntp/)                   |
| `certificate`  | [Certificate](./certificate/)   |
| `certificate_providers` | [Certificate Provider](./shared/certificate-provider/) |
| `http_clients` | [HTTP Client](./shared/http-client/) |
| `network_namespaces` | [Network Namespace](./network-namespace/) |
| `endpoints`    | [Endpoint](./endpoint/)         |
| `inbounds`     | [Inbound](./inbound/)           |
| `outbounds`    | [Outbound](./outbound/)         |
| `route`        | [Route](./route/)               |
| `services`     | [Service](./service/)           |
| `experimental` | [Experimental](./experimental/) |

### Check

```bash
sing-box check
```

### Format

```bash
sing-box format -w -c config.json -D config_directory
```

### Merge

```bash
sing-box merge output.json -c config.json -D config_directory
```

### YAML Support

YAML configuration files share the identical schema and validation rules with JSON.
YAML anchors (`&`), aliases (`*`), and merge keys (`<<`) are supported.

When using `sing-box format` on a YAML file, the output will remain in YAML format
(map keys are sorted alphabetically and comments are not preserved).
When `sing-box merge` writes to a path ending in `.yaml` or `.yml`, the merged configuration
is encoded as YAML.

Rule-set files and other external resources remain JSON or their respective binary formats.
