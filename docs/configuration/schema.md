---
icon: material/new-box
---

# JSON Schema

!!! question "Since sing-box 1.14.0"

sing-box provides a JSON Schema Draft 2020-12 for configuration files.
Compatible editors can use it for completion and validation.

```{.yaml linenums="1"}
$schema: https://sing-box.sagernet.org/schema.json
```

## $schema

The schema URI used by compatible editors.

The schema published with this documentation is available at
[sing-box.sagernet.org/schema.json](https://sing-box.sagernet.org/schema.json).

## Generate

Use the following command to generate a schema matching the installed binary:

```bash
sing-box schema -o schema.json
```

Without `--output`, the schema is written to standard output.
The generated schema reflects the features included in the current build.

You can then reference the local schema from a configuration file:

```{.yaml linenums="1"}
$schema: ./schema.json
```
