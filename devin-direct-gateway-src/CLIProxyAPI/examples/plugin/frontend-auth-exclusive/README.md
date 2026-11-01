# Frontend Auth Exclusive Plugin Example

[English](README.md) | [简体中文](README_CN.md)

This example registers a frontend auth provider with `frontend_auth_provider_exclusive: true`.

When enabled and selected, this provider becomes the only request authentication provider. Built-in config API keys and other frontend auth providers do not authenticate requests while this provider is active.

The example accepts requests that include:

```http
X-Example-Frontend-Auth: exclusive
```

The provider identifier reported to the host is `example-frontend-auth-exclusive-go`.

## Limitations

Token and signature verification is not implemented. The example only compares the header value to `exclusive`, so any client that sends this header authenticates.

## Build

```bash
cd examples/plugin/frontend-auth-exclusive/go
go build -buildmode=c-shared -o /tmp/cliproxy-frontend-auth-exclusive.dylib .
```

Use `.so` on Linux or FreeBSD and `.dll` on Windows. The output filename is the plugin ID, so name the artifact after the key you use under `plugins.configs`.
