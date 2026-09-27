# rule-set

=== "Inline"

    ```{.yaml linenums="1"}
    type: inline  # optional
    tag: ""
    rules: []
    ```

=== "Local File"

    ```{.yaml linenums="1"}
    type: local
    tag: ""  # or []
    format: source  # or binary
    path: ""
    ```

=== "Remote File"

    !!! info ""
    
        Remote rule-set will be cached if `experimental.cache_file.enabled`.

    ```{.yaml linenums="1"}
    type: remote
    tag: ""  # or []
    format: source  # or binary
    url: ""
    initial_path: ""
    http_client: ""  # or {}
    update_interval: ""
    ```

## type

**Required.** Type of rule-set, `local` or `remote`.

## tag

**Required.** Tag of rule-set.

## Inline Fields

### rules

**Required.** List of [Headless Rule](./headless-rule/).

## Local or Remote Fields

### format

**Required.** Format of rule-set file, `source` or `binary`.

Optional when `path` or `url` uses `json` or `srs` as extension.

## Local Fields

### path

**Required.**

!!! note ""

    Will be automatically reloaded if file modified since sing-box 1.10.0.

File path of rule-set.

## Remote Fields

### url

**Required.** Download URL of rule-set.

### initial_path

File path of the initial rule-set content.

Used at startup when no cached rule-set is available.

### http_client

HTTP Client for downloading rule-set.

See [HTTP Client Fields](/configuration/shared/http-client/) for details.

When empty, the default HTTP client is used: the one named by
[`default_http_client`](/configuration/route/#default_http_client), or the first top-level
`http_clients` entry when `default_http_client` is empty.

### update_interval

Update interval of rule-set.

`1d` will be used if empty.
