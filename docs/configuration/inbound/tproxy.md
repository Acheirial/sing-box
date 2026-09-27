# TProxy

!!! quote ""

    Only supported on Linux.

```{.yaml linenums="1"}
type: tproxy
tag: tproxy-in

# ... Listen Fields

network: udp

# ... UDP NAT Fields

```

## Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

## network

Listen network, one of `tcp` `udp`.

Both if empty.

## UDP NAT Fields

See [UDP NAT Fields](/configuration/shared/udp-nat/) for details.
