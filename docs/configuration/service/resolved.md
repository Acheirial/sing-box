# Resolved

Resolved service is a fake systemd-resolved DBUS service to receive DNS settings from other programs
(e.g. NetworkManager) and provide DNS resolution.

See also: [Resolved DNS Server](/configuration/dns/server/resolved/)

```{.yaml linenums="1"}
type: resolved

# ... Listen Fields

```

## Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

## listen

**Required.** Listen address.

`127.0.0.53` will be used by default.

## listen_port

**Required.** Listen port.

`53` will be used by default.
