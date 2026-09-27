# Clash API

=== "结构"

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

=== "示例 (在线)"

    ```{.yaml linenums="1"}
    external_controller: 127.0.0.1:9090
    access_control_allow_origin:
      - http://127.0.0.1
      - http://yacd.haishan.me
    access_control_allow_private_network: true
    ```

=== "示例 (下载)"

    ```{.yaml linenums="1"}
    external_controller: 0.0.0.0:9090
    external_ui: dashboard
    # external_ui_download_detour: direct
    ```

!!! note ""

    当内容只有一项时，可以直接使用单个值，无需数组

## external_controller

RESTful web API 监听地址。如果为空，则禁用 Clash API。

## external_ui

到静态网页资源目录的相对路径或绝对路径。sing-box 会在 `http://{{external-controller}}/ui` 下提供它。

## external_ui_download_url

静态网页资源的 ZIP 下载 URL，如果指定的 `external_ui` 目录为空，将使用。

默认使用 `https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip`。

## external_ui_download_detour

用于下载静态网页资源的出站的标签。

如果为空，将使用默认出站。

## secret

RESTful API 的密钥（可选）
通过指定 HTTP 标头 `Authorization: Bearer ${secret}` 进行身份验证
如果 RESTful API 正在监听 0.0.0.0，请始终设置一个密钥。

## default_mode

Clash 中的默认模式，默认使用 `Rule`。

此设置没有直接影响，但可以通过 `clash_mode` 规则项在路由和 DNS 规则中使用。

## access_control_allow_origin

允许的 CORS 来源，默认使用 `*`。

要从公共网站访问私有网络上的 Clash API，必须在 `access_control_allow_origin` 中明确指定它而不是使用 `*`。

## access_control_allow_private_network

允许从私有网络访问。

要从公共网站访问私有网络上的 Clash API，必须启用 `access_control_allow_private_network`。
