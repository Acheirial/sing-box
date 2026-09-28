# TLS

## Inbound

```{.yaml linenums="1"}
enabled: true
server_name: ""
alpn: []
min_version: ""
max_version: ""
cipher_suites: []
curve_preferences: []
certificate: []
certificate_path: ""
client_authentication: ""
client_certificate: []
client_certificate_path: []
client_certificate_sha256: []
client_certificate_public_key_sha256: []
key: []
key_path: ""
kernel_tx: false
kernel_rx: false
handshake_timeout: ""
certificate_provider: ""

ech:
  enabled: false
  key: []
  key_path: ""
reality:
  enabled: false
  handshake:
    server: google.com
    server_port: 443

    # ... Dial Fields
  private_key: UuMBgl7MXTPx9inmQp2UC7Jcnwc6XYbwDNebonM-FCc
  short_id:
    - 0123456789abcdef
  max_time_difference: 1m
  mldsa65_seed: ""
```

## Outbound

```{.yaml linenums="1"}
enabled: true
engine: ""
disable_sni: false
server_name: ""
insecure: false
alpn: []
min_version: ""
max_version: ""
cipher_suites: []
curve_preferences: []
certificate: ""
certificate_path: ""
certificate_sha256: []
certificate_public_key_sha256: []
client_certificate: []
client_certificate_path: ""
client_key: []
client_key_path: ""
fragment: false
fragment_fallback_delay: ""
record_fragment: false
spoof: ""
spoof_method: ""
kernel_tx: false
kernel_rx: false
handshake_timeout: ""
ech:
  enabled: false
  config: []
  config_path: ""
  query_server_name: ""
utls:
  enabled: false
  fingerprint: ""
reality:
  enabled: false
  public_key: jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0
  short_id: 0123456789abcdef
  mldsa65_verify: ""
```

TLS version values:

* `1.0`
* `1.1`
* `1.2`
* `1.3`

Cipher suite values:

* `TLS_RSA_WITH_AES_128_CBC_SHA`
* `TLS_RSA_WITH_AES_256_CBC_SHA`
* `TLS_RSA_WITH_AES_128_GCM_SHA256`
* `TLS_RSA_WITH_AES_256_GCM_SHA384`
* `TLS_AES_128_GCM_SHA256`
* `TLS_AES_256_GCM_SHA384`
* `TLS_CHACHA20_POLY1305_SHA256`
* `TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA`
* `TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA`
* `TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA`
* `TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA`
* `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`
* `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`
* `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
* `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`
* `TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256`
* `TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256`

!!! note ""

    You can use a single value instead of an array when the content is only one item

## enabled

Enable TLS.

## engine

**Client only.** TLS engine to use.

Values:

* `go` (default)
* `apple`
* `windows`

Supported fields:

* `server_name`
* `insecure`
* `alpn`
* `min_version`
* `max_version`
* `certificate` / `certificate_path`
* `certificate_sha256`
* `certificate_public_key_sha256`
* `handshake_timeout`

Unsupported fields:

* `disable_sni`
* `cipher_suites`
* `curve_preferences`
* `client_certificate` / `client_certificate_path` / `client_key` / `client_key_path`
* `fragment` / `record_fragment`
* `kernel_tx` / `kernel_rx`
* `ech`
* `utls`
* `reality`

!!! note ""

    `windows` uses Schannel via SSPI. Only available on Windows build 17763 or later (Windows 10 version 1809, Windows Server 2019, or newer).

!!! note ""

    TLS 1.3 is only negotiated on Windows 11 or Windows Server 2022 and newer.

The default version range is TLS 1.2 to TLS 1.3, matching the `go` engine.

## disable_sni

**Client only.** Do not send server name in ClientHello.

## server_name

Used to verify the hostname on the returned certificates unless insecure is given.

It is also included in the client's handshake to support virtual hosting unless it is an IP address.

## insecure

**Client only.** Accepts any server certificate.

## alpn

List of supported application level protocols, in order of preference.

If both peers support ALPN, the selected protocol will be one from this list, and the connection will fail if there is
no mutually supported protocol.

See [Application-Layer Protocol Negotiation](https://en.wikipedia.org/wiki/Application-Layer_Protocol_Negotiation).

## min_version

The minimum TLS version that is acceptable.

TLS 1.2 is used by default.

## max_version

The maximum TLS version that is acceptable.

By default, the maximum version is currently TLS 1.3.

## cipher_suites

List of enabled TLS 1.0–1.2 cipher suites. The order of the list is ignored.
Note that TLS 1.3 cipher suites are not configurable.

If empty, a safe default list is used. The default cipher suites might change over time.

## curve_preferences

Set of supported key exchange mechanisms. The order of the list is ignored, and key exchange mechanisms are chosen
from this list using an internal preference order by Golang.

Available values, also the default list:

* `P256`
* `P384`
* `P521`
* `X25519`
* `X25519MLKEM768`

## certificate

Server certificates chain line array, in PEM format.

## certificate_path

!!! note ""

    Will be automatically reloaded if file modified.

The path to server certificate chain, in PEM format.

## certificate_sha256

**Client only.** List of SHA-256 hashes of server certificates, in base64 format.

The hash is computed over the whole DER-encoded certificate. Use `certificate_public_key_sha256` to pin only the public key.

To generate the SHA-256 hash for a certificate, use the following commands:

```bash
# For a certificate file
openssl x509 -in certificate.pem -outform der | openssl dgst -sha256 -binary | openssl enc -base64

# For a certificate from a remote server
echo | openssl s_client -servername example.com -connect example.com:443 2>/dev/null | openssl x509 -outform der | openssl dgst -sha256 -binary | openssl enc -base64
```

## certificate_public_key_sha256

**Client only.** List of SHA-256 hashes of server certificate public keys, in base64 format.

To generate the SHA-256 hash for a certificate's public key, use the following commands:

```bash

# For a certificate file

openssl x509 -in certificate.pem -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64

# For a certificate from a remote server

echo | openssl s_client -servername example.com -connect example.com:443 2>/dev/null | openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64
```

## client_certificate

**Client only.** Client certificate chain line array, in PEM format.

## client_certificate_path

**Client only.** The path to client certificate chain, in PEM format.

## client_key

**Client only.** Client private key line array, in PEM format.

## client_key_path

**Client only.** The path to client private key, in PEM format.

## key

**Server only.** The server private key line array, in PEM format.

## key_path

**Server only.**

!!! note ""

    Will be automatically reloaded if file modified.

The path to the server private key, in PEM format.

## client_authentication

**Server only.** The type of client authentication to use.

Available values:

* `no` (default)
* `request`
* `require-any`
* `verify-if-given`
* `require-and-verify`

One of `client_certificate`, `client_certificate_path`, `client_certificate_sha256`, or `client_certificate_public_key_sha256` is required
if this option is set to `verify-if-given`, or `require-and-verify`.

## client_certificate

**Server only.** Client certificate chain line array, in PEM format.

## client_certificate_path

**Server only.**

!!! note ""

    Will be automatically reloaded if file modified.

List of path to client certificate chain, in PEM format.

## client_certificate_sha256

**Server only.** List of SHA-256 hashes of client certificates, in base64 format.

The hash is computed over the whole DER-encoded certificate, see [certificate_sha256](#certificate_sha256).

## client_certificate_public_key_sha256

**Server only.** List of SHA-256 hashes of client certificate public keys, in base64 format.

To generate the SHA-256 hash for a certificate's public key, use the following commands:

```bash

# For a certificate file

openssl x509 -in certificate.pem -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64

# For a certificate from a remote server

echo | openssl s_client -servername example.com -connect example.com:443 2>/dev/null | openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64
```

## kernel_tx

!!! quote ""

    Only supported on Linux 5.1+, use a newer kernel if possible.

!!! quote ""

    Only TLS 1.3 is supported.

!!! warning ""

    kTLS TX may only improve performance when `splice(2)` is available (both ends must be TCP or TLS without additional protocols after handshake); otherwise, it will definitely degrade performance.

Enable kernel TLS transmit support.

## kernel_rx

!!! quote ""

    Only supported on Linux 5.1+, use a newer kernel if possible.

!!! quote ""

    Only TLS 1.3 is supported.

!!! failure ""

    kTLS RX will definitely degrade performance even if `splice(2)` is in use, so enabling it is not recommended.

Enable kernel TLS receive support.

## handshake_timeout

TLS handshake timeout, in golang's Duration format.

`15s` is used by default.

## certificate_provider

**Server only.** A string or an object.

When string, the tag of a shared [Certificate Provider](/configuration/shared/certificate-provider/).

When object, an inline certificate provider. See [Certificate Provider](/configuration/shared/certificate-provider/) for available types and fields.

## Custom TLS support

!!! info "QUIC support"

    Only ECH is supported in QUIC.

### utls

**Client only.**

!!! failure "Not Recommended"

    uTLS has had repeated fingerprinting vulnerabilities discovered by researchers.

    uTLS is a Go library that attempts to imitate browser TLS fingerprints by copying
    ClientHello structure. However, browsers use completely different TLS stacks
    (Chrome uses BoringSSL, Firefox uses NSS) with distinct implementation behaviors
    that cannot be replicated by simply copying the handshake format, making detection possible.
    Additionally, the library lacks active maintenance and has poor code quality,
    making it unsuitable for censorship circumvention.

    For TLS fingerprint resistance, use [NaiveProxy](/configuration/inbound/naive/) instead.

uTLS is a fork of "crypto/tls", which provides ClientHello fingerprinting resistance.

Available fingerprint values:

* chrome
* firefox
* edge
* safari
* 360
* qq
* ios
* android
* random
* randomized

Chrome fingerprint will be used if empty.

## ECH Fields

ECH (Encrypted Client Hello) is a TLS extension that allows a client to encrypt the first part of its ClientHello
message.

The ECH key and configuration can be generated by `sing-box generate ech-keypair`.

### key

**Server only.** ECH key line array, in PEM format.

### key_path

**Server only.**

!!! note ""

    Will be automatically reloaded if file modified.

The path to ECH key, in PEM format.

### config

**Client only.** ECH configuration line array, in PEM format.

If empty, load from DNS will be attempted.

### config_path

**Client only.** The path to ECH configuration, in PEM format.

If empty, load from DNS will be attempted.

### query_server_name

**Client only.** Overrides the domain name used for ECH HTTPS record queries.

If empty, `server_name` is used for queries.

### fragment

**Client only.** Fragment TLS handshakes to bypass firewalls.

This feature is intended to circumvent simple firewalls based on **plaintext packet matching**,
and should not be used to circumvent real censorship.

Due to poor performance, try `record_fragment` first, and only apply to server names known to be blocked.

On Linux, Apple platforms, (administrator privileges required) Windows,
the wait time can be automatically detected. Otherwise, it will fall back to
waiting for a fixed time specified by `fragment_fallback_delay`.

In addition, if the actual wait time is less than 20ms, it will also fall back to waiting for a fixed time,
because the target is considered to be local or behind a transparent proxy.

### fragment_fallback_delay

**Client only.** The fallback value used when TLS segmentation cannot automatically determine the wait time.

`500ms` is used by default.

### record_fragment

**Client only.** Fragment TLS handshake into multiple TLS records to bypass firewalls.

### spoof

**Client only.**

!!! quote ""

    Only supported on Linux, macOS, and Windows, and requires elevated privileges.

Inject a forged TLS ClientHello carrying a whitelisted SNI before the real one,
to fool SNI-filtering middleboxes that permit specific hostnames.

Requires `CAP_NET_RAW` and `CAP_NET_ADMIN` on Linux, root on macOS, and
Administrator on Windows. Windows on ARM64 is not supported.

### spoof_method

**Client only.** How the forged segment is rejected by the real server.

| Value                      | Behavior                                                                                                       |
|----------------------------|----------------------------------------------------------------------------------------------------------------|
| `wrong-sequence` (default) | The forged segment's TCP sequence number is placed before the server's receive window.                         |
| `wrong-checksum`           | The forged segment's TCP checksum is deliberately invalid.                                                     |
| `wrong-ack`                | The forged segment's TCP acknowledgment number is placed before the server's send window.                      |
| `wrong-md5`                | The forged segment carries a TCP-MD5 signature option.                                                         |
| `wrong-timestamp`          | The forged segment carries a backdated TCP timestamp. Linux/Windows only; not supported on macOS.              |

## Reality Fields

### handshake

**Server only.**

**Required.** Handshake server address and [Dial Fields](/configuration/shared/dial/).

### private_key

**Server only.**

**Required.** Private key, generated by `sing-box generate reality-keypair`.

### server_names

**Server only.** Additional TLS server names (SNI) accepted by the server,
besides `server_name`. Empty by default.

### min_client_ver

**Server only.** Minimum accepted REALITY client version, encoded as
`major.minor.patch`. Empty disables the bound.

### max_client_ver

**Server only.** Maximum accepted REALITY client version, encoded as
`major.minor.patch`. Empty disables the bound.

### limit_fallback_upload

**Server only.** Rate limit applied to data uploaded over the fallback
connection:

* `after_bytes`: transferred bytes allowed before limiting starts.
* `bytes_per_sec`: sustained rate.
* `burst_bytes_per_sec`: burst rate.

### limit_fallback_download

**Server only.** Rate limit applied to data downloaded over the fallback
connection, with the same fields as `limit_fallback_upload`.

### xver

**Server only.** PROXY protocol version (`0`, `1` or `2`) requested for the
fallback connection. When configured, PROXY protocol (v1 or v2) is emitted on the fallback connection.

### mldsa65_seed

**Server only.** Base64 URL-encoded 32-byte seed for ML-DSA-65 server certificate signing.

### show

**Server only.** Log REALITY handshake details at info level instead of trace
level. Disabled by default.

### master_key_log

**Server only.** File that receives TLS master secrets in NSS key log format.
`none` disables it.

### public_key

**Client only.**

**Required.** Public key, generated by `sing-box generate reality-keypair`.

### short_id

**Required.** A hexadecimal string with zero to eight digits.

### max_time_difference

**Server only.** The maximum time difference between the server and the client.

Check disabled if empty.

### mldsa65_verify

**Client only.** A 1952 bytes ML-DSA-65 public key in base64 format, used to verify the additional post-quantum signature in the first certificate extension.

### spider_x

**Client only.** Initial path used when crawling the fallback server after a
failed verification. The `p`, `c`, `t`, `i` and `r` query parameters configure
SpiderY; see the Xray-core REALITY documentation for their meaning.
