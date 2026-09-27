---
icon: material/new-box
---

# Hosts

!!! question "Since sing-box 1.12.0"

```{.yaml linenums="1"}
dns:
  servers:
    - type: hosts
      tag: ""

      path: []
      predefined: {}
```

!!! note ""

    You can use a single value instead of an array when the content is only one item

## path

List of paths to hosts files.

`/etc/hosts` is used by default.

`C:\Windows\System32\Drivers\etc\hosts` is used by default on Windows.

Example:

```{.yaml linenums="1"}

# path: /etc/hosts

path:
  - /etc/hosts
  - $HOME/.hosts
```

## predefined

Predefined hosts.

Example:

```{.yaml linenums="1"}
predefined:
  www.google.com: 127.0.0.1
  localhost:
    - 127.0.0.1
    - "::1"
```

## Examples

=== "Use hosts if available"

    === ":material-card-multiple: sing-box 1.14.0"

        ```{.yaml linenums="1"}
        dns:
          servers:
            # ...
            - type: hosts
              tag: hosts
          rules:
            - preferred_by: hosts
              action: route
              server: hosts
        ```

    === ":material-card-remove: sing-box < 1.14.0"

        ```{.yaml linenums="1"}
        dns:
          servers:
            # ...
            - type: hosts
              tag: hosts
          rules:
            - ip_accept_any: true
              server: hosts
        ```
