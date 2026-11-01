# Codex Service Tier 插件

[English](README.md) | [简体中文](README_CN.md)

这是一个针对 Codex 出站请求的请求规范化插件。

插件启用且 `fast` 设置为 `true` 时，对同时满足以下条件的请求，它会把顶层 `service_tier` 字段设置为 `priority`：

- `req.ToFormat` 为 `codex`
- `req.Model` 为 `gpt-5.5`

插件暂不支持其他模型或格式，这类请求原样返回。

## 配置

在 `plugins.configs` 下添加插件配置：

```yaml
plugins:
  configs:
    codex-service-tier:
      enabled: true
      priority: 1
      fast: false
```

`fast` 是布尔字段。设为 `true` 后，匹配的 Codex `gpt-5.5` 请求会被改写为 priority 服务层级。

## 构建

```bash
cd go
go build -buildmode=c-shared -o /tmp/codex-service-tier.dylib .
rm -f /tmp/codex-service-tier.h
```

Linux 或 FreeBSD 使用 `.so`，Windows 使用 `.dll`。输出文件名即插件 ID，因此要配合上面的配置，产物必须命名为 `codex-service-tier`。
