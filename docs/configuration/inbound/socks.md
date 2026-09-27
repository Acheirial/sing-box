# SOCKS

`socks` inbound is a socks4, socks4a, socks5 server.

```{.yaml linenums="1"}
type: socks
tag: socks-in

# ... Listen Fields

users:
  - username: admin
    password: admin
```

## Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

## users

SOCKS users.

No authentication required if empty.
