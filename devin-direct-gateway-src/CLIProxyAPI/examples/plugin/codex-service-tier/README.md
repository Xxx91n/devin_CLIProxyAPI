# Codex Service Tier Plugin

[English](README.md) | [简体中文](README_CN.md)

This plugin is a request normalizer for Codex outbound requests.

When the plugin is enabled and `fast` is set to `true`, it sets the top-level `service_tier` field to `priority` for requests where:

- `req.ToFormat` is `codex`
- `req.Model` is `gpt-5.5`

Requests for other models or formats are out of scope and are returned unchanged.

## Configuration

Add the plugin under `plugins.configs`:

```yaml
plugins:
  configs:
    codex-service-tier:
      enabled: true
      priority: 1
      fast: false
```

`fast` is a boolean field. Set it to `true` to enable priority service tier shaping for matching Codex `gpt-5.5` requests.

## Build

```bash
cd go
go build -buildmode=c-shared -o /tmp/codex-service-tier.dylib .
rm -f /tmp/codex-service-tier.h
```

Use `.so` on Linux or FreeBSD and `.dll` on Windows. The output filename is the plugin ID, so the artifact must be named `codex-service-tier` for the configuration above.
