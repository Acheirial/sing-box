# 规则集

=== "内联"

    ```{.yaml linenums="1"}
    type: inline  # 可选
    tag: ""
    rules: []
    ```

=== "本地文件"

    ```{.yaml linenums="1"}
    type: local
    tag: ""  # 或 []
    format: source  # or binary
    path: ""
    ```

=== "远程文件"

    !!! info ""
    
        远程规则集将被缓存如果 `experimental.cache_file.enabled` 已启用。

    ```{.yaml linenums="1"}
    type: remote
    tag: ""  # 或 []
    format: source  # or binary
    url: ""
    initial_path: ""
    http_client: ""  # 或 {}
    update_interval: ""
    ```

## type

**必填。**规则集类型， `local` 或 `remote`。

## tag

**必填。**规则集的标签。

## 内联字段

### rules

**必填。**一组 [无头规则](./headless-rule/).

## 本地或远程字段

### format

**必填。**规则集格式， `source` 或 `binary`。

当 `path` 或 `url` 使用 `json` 或 `srs` 作为扩展名时可选。

## 本地字段

### path

**必填。**

!!! note ""

    自 sing-box 1.10.0 起，文件更改时将自动重新加载。

规则集的文件路径。

## 远程字段

### url

**必填。**规则集的下载 URL。

### initial_path

规则集初始内容的文件路径。

在启动时没有可用的规则集缓存时使用。

### http_client

用于下载规则集的 HTTP 客户端。

参阅 [HTTP 客户端字段](/zh/configuration/shared/http-client/) 了解详情。

留空时使用默认 HTTP 客户端：即由 [`default_http_client`](/zh/configuration/route/#default_http_client)
指定的客户端，或当 `default_http_client` 为空时使用顶级 `http_clients` 的第一项。

### update_interval

规则集的更新间隔。

默认使用 `1d`。
