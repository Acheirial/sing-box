# DNS01 Challenge Fields

```{.yaml linenums="1"}
ttl: ""
propagation_delay: ""
propagation_timeout: ""
resolvers: []
override_domain: ""
provider: ""

# ... Provider Fields

```

## ttl

The TTL of the temporary TXT record used for the DNS challenge.

## propagation_delay

How long to wait after creating the challenge record before starting propagation checks.

## propagation_timeout

The maximum time to wait for the challenge record to propagate.

Set to `-1` to disable propagation checks.

## resolvers

Preferred DNS resolvers to use for DNS propagation checks.

## override_domain

Override the domain name used for the DNS challenge record.

Useful when `_acme-challenge` is delegated to a different zone.

## provider

The DNS provider. See below for provider-specific fields.

## Provider Fields

### Alibaba Cloud DNS

```{.yaml linenums="1"}
provider: alidns
access_key_id: ""
access_key_secret: ""
region_id: ""
security_token: ""
```

#### security_token

The Security Token for STS temporary credentials.

### Cloudflare

```{.yaml linenums="1"}
provider: cloudflare
api_token: ""
zone_token: ""
```

#### zone_token

Optional API token with `Zone:Read` permission.

When provided, allows `api_token` to be scoped to a single zone.

### ACME-DNS

```{.yaml linenums="1"}
provider: acmedns
username: ""
password: ""
subdomain: ""
server_url: ""
```

See [ACME-DNS](https://github.com/joohoi/acme-dns) for details.
