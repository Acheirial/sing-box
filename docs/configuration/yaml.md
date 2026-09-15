---
icon: material/new-box
---

!!! question "Since sing-box 1.15.0"

# YAML

sing-box accepts YAML configuration files with a `.yaml` or `.yml` file extension, in addition to the default JSON format.

### How it works

YAML configuration files are parsed and converted to JSON internally, then processed like any other configuration.

All configuration options are identical to the JSON format. The file extension determines the parser: files ending in `.yaml` or `.yml` are treated as YAML, everything else as JSON.

### Example

The following YAML configuration is equivalent to a basic JSON configuration:

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

### Limitations

* Configuration read from standard input (`-c stdin`) must be in JSON format.
* Configuration provided via library APIs (libbox/daemon) remains JSON-only.
* `sing-box format -w` writes formatted JSON back to the file. Since YAML is converted on load, the file on disk becomes JSON after formatting.
