# Introduction

sing-box configuration files are written in YAML.
Files ending in `.yaml`, `.yml` and `.json` are automatically recognized
by the `-c/--config` and `-C/--config-directory` options.
For `stdin` and unrecognized extensions, the format is detected from the content.

YAML anchors (`&`), aliases (`*`) and merge keys (`<<`) are supported.

### Structure

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
sing-box format -w -c config.yaml -D config_directory
```

The output stays in YAML format, with map keys sorted alphabetically.
Comments are not preserved.

### Merge

```bash
sing-box merge output.yaml -c config.yaml -D config_directory
```

### JSON Support

JSON configuration files are also supported and share the identical schema and
validation rules with YAML.

Rule-set files and other external resources remain JSON or their respective binary formats.
