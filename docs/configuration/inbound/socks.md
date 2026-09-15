`socks` inbound is a socks4, socks4a, socks5 server.

### Structure

```yaml
type: socks
tag: socks-in
users:
- username: admin
  password: admin
```

### Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

### Fields

#### users

SOCKS users.

No authentication required if empty.
