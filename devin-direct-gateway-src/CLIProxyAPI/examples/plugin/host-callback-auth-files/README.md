# Host callback auth files plugin

[English](README.md) | [简体中文](README_CN.md)

A Go-only example plugin whose Management API page calls the host auth file callbacks:

- `host.auth.list`
- `host.auth.get`
- `host.auth.get_runtime`
- `host.auth.save`

## Purpose and scope

The plugin registers a Management API resource named `Host Auth Files` at `/status`, exposed under:

```text
/v0/resource/plugins/host-callback-auth-files/status
```

The page reads URL query parameters, calls the host auth callbacks, and renders the result as HTML. It registers no executor, translator, auth provider, or scheduler capability.

## Build

From this directory:

```bash
cd go
go build -buildmode=c-shared -o host-callback-auth-files.dylib .
rm -f host-callback-auth-files.h
cp host-callback-auth-files.dylib ../../plugins/
```

Use the extension your target system expects: `.dylib` on macOS, `.so` on Linux, `.dll` on Windows.

## Configuration

The library goes into the configured plugin directory (`plugins.dir`, default `plugins`) with a basename matching the plugin ID: `plugins/host-callback-auth-files.dylib` maps to `plugins.configs.host-callback-auth-files`.

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    host-callback-auth-files:
      enabled: true
      priority: 1
```

The plugin defines no configuration fields of its own.

## Resource URL examples

List all auth files:

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=list
```

Read physical JSON by auth index:

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=get&auth_index=<AUTH_INDEX>
```

Read runtime info by auth index:

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=runtime&auth_index=<AUTH_INDEX>
```

Save physical JSON:

```text
http://localhost:8080/v0/resource/plugins/host-callback-auth-files/status?op=save&name=example-auth.json&json=%7B%22type%22%3A%22gemini%22%2C%22email%22%3A%22demo%40example.com%22%2C%22api_key%22%3A%22demo-key%22%7D
```

## Parameters

| Parameter | Description |
| --- | --- |
| `op` | One of `list` (default), `get`, `runtime` (alias `get_runtime`), `save`. |
| `auth_index` | Required for `get` and `runtime`. |
| `name` | Required for `save`. Must end with `.json`. |
| `json` | Required for `save`. Must be valid JSON. |

Ops other than `list`, `get`, `runtime`, and `save` are not supported; the page reports an error for them.

## Notes

- `host.auth.get` returns the physical auth file JSON.
- `host.auth.get_runtime` returns runtime credential metadata.
- `host.auth.save` writes the JSON to the auth directory and upserts the runtime auth record.
