# Scheduler Plugin

[English](README.md) | [简体中文](README_CN.md)

This plugin demonstrates the CLIProxyAPI C ABI scheduler capability from Go.

It implements:

- `plugin.register`
- `plugin.reconfigure`
- `scheduler.pick`

The plugin can select a configured auth ID, delegate routing to a built-in scheduler, or reject scheduler picks.

## Configuration

Add the plugin under `plugins.configs`:

```yaml
plugins:
  configs:
    scheduler:
      enabled: true
      priority: 1
      auth_id: ""
      delegate: ""
      deny: false
```

Fields:

- `auth_id`: selects this auth ID when it appears in the scheduler candidates.
- `delegate`: delegates selection to a built-in scheduler. Supported values are `""`, `fill-first`, and `round-robin`.
- `deny`: returns a scheduler error when set to `true`.

Behavior:

- When `deny` is `true`, the plugin returns an error envelope with code `scheduler_denied`.
- When `delegate` is `fill-first` or `round-robin`, the plugin returns `DelegateBuiltin` and marks the pick as handled.
- When `delegate` is any other non-empty value, the plugin leaves the pick unhandled.
- When `delegate` is empty and `auth_id` exists in the candidates, the plugin returns that auth ID and marks the pick as handled.
- When no rule matches, the plugin leaves the pick unhandled.

## Limitations

`deny` takes precedence over `auth_id` and `delegate`. When it is `true`, the other two fields have no effect.

## Build

From this directory:

```bash
cd go
go build -buildmode=c-shared -o /tmp/cliproxy-scheduler-plugin.so .
rm -f /tmp/cliproxy-scheduler-plugin.h
```

The output filename is the plugin ID, so name the artifact after the key you use under `plugins.configs`.
