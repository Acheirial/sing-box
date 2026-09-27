# 源文件格式

```json
{
  "version": 3,
  "rules": []
}
```

## 编译

使用 `sing-box rule-set compile [--output <file-name>.srs] <file-name>.json` 以编译源文件为二进制规则集。

## version

**必填。**规则集版本。

* 1: sing-box 1.8.0: 初始规则集版本。
* 2: sing-box 1.10.0: 优化了二进制规则集中 `domain_suffix` 规则的内存使用。
* 3: sing-box 1.11.0: 添加了 `network_type`、 `network_is_expensive` 和 `network_is_constrainted` 规则项。
* 4: sing-box 1.13.0: 添加了 `network_interface_address` 和 `default_interface_address` 规则项。
* 5: sing-box 1.14.0: 添加了 `package_name_regex` 规则项。

## rules

**必填。**一组 [无头规则](/zh/configuration/rule-set/headless-rule/).
