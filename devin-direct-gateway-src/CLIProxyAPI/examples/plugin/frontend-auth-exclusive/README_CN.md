# Frontend Auth 独占插件示例

[English](README.md) | [简体中文](README_CN.md)

这个示例注册了一个带有 `frontend_auth_provider_exclusive: true` 的前端认证提供方。

插件启用并被选中后，该提供方成为唯一的请求认证提供方。它生效期间，内置配置 API key 和其他前端认证提供方都无法认证请求。

示例只接受包含以下请求头的请求：

```http
X-Example-Frontend-Auth: exclusive
```

插件上报给宿主的提供方标识是 `example-frontend-auth-exclusive-go`。

## 局限

示例不校验 token 或签名，只把请求头的值与 `exclusive` 做比较。任何发送这个请求头的客户端都能通过认证。

## 构建

```bash
cd examples/plugin/frontend-auth-exclusive/go
go build -buildmode=c-shared -o /tmp/cliproxy-frontend-auth-exclusive.dylib .
```

Linux 或 FreeBSD 使用 `.so`，Windows 使用 `.dll`。输出文件名即插件 ID，请按 `plugins.configs` 中使用的配置键命名产物。
