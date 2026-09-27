# Tailscale

```{.yaml linenums="1"}
dns:
  servers:
    - type: tailscale
      tag: ""

      endpoint: ts-ep
      accept_default_resolvers: false
      accept_search_domain: false
```

## endpoint

**Required.** The tag of the [Tailscale Endpoint](/configuration/endpoint/tailscale).

## accept_default_resolvers

Indicates whether default DNS resolvers should be accepted for fallback queries in addition to MagicDNS。

if not enabled, `NXDOMAIN` will be returned for non-Tailscale domain queries.

## accept_search_domain

When enabled, single-label queries (e.g. `my-device`) are retried against each Tailscale search domain until one resolves.

## Examples

=== "MagicDNS only"

    === ":material-card-multiple: sing-box 1.14.0"

        ```{.yaml linenums="1"}
        dns:
          servers:
            - type: local
              tag: local
            - type: tailscale
              tag: ts
              endpoint: ts-ep
          rules:
            - preferred_by: ts
              action: route
              server: ts
        ```

    === ":material-card-remove: sing-box < 1.14.0"

        ```{.yaml linenums="1"}
        dns:
          servers:
            - type: local
              tag: local
            - type: tailscale
              tag: ts
              endpoint: ts-ep
          rules:
            - ip_accept_any: true
              server: ts
        ```

=== "Use as global DNS"

    ```{.yaml linenums="1"}
    dns:
      servers:
        - type: tailscale
          endpoint: ts-ep
          accept_default_resolvers: true
    ```
