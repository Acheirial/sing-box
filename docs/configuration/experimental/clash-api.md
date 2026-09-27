# Clash API

=== "Structure"

    ```{.yaml linenums="1"}
    external_controller: 127.0.0.1:9090
    external_ui: ""
    external_ui_download_url: ""
    external_ui_download_detour: ""
    secret: ""
    default_mode: ""
    access_control_allow_origin: []
    access_control_allow_private_network: false
    ```

=== "Example (online)"

    ```{.yaml linenums="1"}
    external_controller: 127.0.0.1:9090
    access_control_allow_origin:
      - http://127.0.0.1
      - http://yacd.haishan.me
    access_control_allow_private_network: true
    ```

=== "Example (download)"

    ```{.yaml linenums="1"}
    external_controller: 0.0.0.0:9090
    external_ui: dashboard
    # external_ui_download_detour: direct
    ```

!!! note ""

    You can use a single value instead of an array when the content is only one item

## external_controller

RESTful web API listening address. Clash API will be disabled if empty.

## external_ui

A relative path to the configuration directory or an absolute path to a
directory in which you put some static web resource. sing-box will then
serve it at `http://{{external-controller}}/ui`.

## external_ui_download_url

ZIP download URL for the external UI, will be used if the specified `external_ui` directory is empty.

`https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip` will be used if empty.

## external_ui_download_detour

The tag of the outbound to download the external UI.

Default outbound will be used if empty.

## secret

Secret for the RESTful API (optional)
Authenticate by spedifying HTTP header `Authorization: Bearer ${secret}`
ALWAYS set a secret if RESTful API is listening on 0.0.0.0

## default_mode

Default mode in clash, `Rule` will be used if empty.

This setting has no direct effect, but can be used in routing and DNS rules via the `clash_mode` rule item.

## access_control_allow_origin

CORS allowed origins, `*` will be used if empty.

To access the Clash API on a private network from a public website, you must explicitly specify it in `access_control_allow_origin` instead of using `*`.

## access_control_allow_private_network

Allow access from private network.

To access the Clash API on a private network from a public website, `access_control_allow_private_network` must be enabled.
