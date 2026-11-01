# Host model callback plugin

[English](README.md) | [简体中文](README_CN.md)

A Go-only example plugin whose Management API page runs model execution through the host model callbacks. The plugin sends no outbound HTTP requests itself.

## Purpose and scope

The plugin registers a Management API resource named `Host Model Callback` at `/status`, exposed under:

```text
/v0/resource/plugins/host-model-callback/status
```

The page is query-driven: it reads URL query parameters, builds an OpenAI-compatible chat request, and calls:

- `host.model.execute` for non-streaming execution.
- `host.model.execute_stream`, `host.model.stream_read`, and `host.model.stream_close` for streaming execution.

The example demonstrates host model callbacks only. It registers no executor, translator, normalizer, auth provider, or scheduler capability, and it contains no outbound HTTP client.

## Build

From this directory:

```bash
cd go
go build -buildmode=c-shared -o host-model-callback.dylib .
rm -f host-model-callback.h
cp host-model-callback.dylib ../../plugins/
```

Use the extension your target system expects: `.dylib` on macOS, `.so` on Linux, `.dll` on Windows.

## Configuration

The library goes into the configured plugin directory (`plugins.dir`, default `plugins`) with a basename matching the plugin ID: `plugins/host-model-callback.dylib` maps to `plugins.configs.host-model-callback`.

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    host-model-callback:
      enabled: true
      priority: 1
```

The plugin defines no configuration fields of its own.

## Resource URL examples

Non-streaming, all defaults:

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status
```

Non-streaming with explicit protocol and prompt:

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?entry_protocol=openai&exit_protocol=openai&model=gpt-5.5&prompt=Say%20hello%20in%20one%20sentence
```

Streaming with explicit close:

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?stream=true&model=gpt-5.5&prompt=Write%20three%20short%20tokens
```

Streaming with RPC-scope implicit close:

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?stream=true&implicit_close=true
```

The default model ID is `gpt-5.5`, matching the nearby `codex-service-tier` example. It is only an example identifier; the request succeeds only when your CPA configuration can route that model.

## Parameters

| Parameter | Description |
| --- | --- |
| `entry_protocol` | Inbound client protocol passed to the host execution path. Default `openai`. |
| `exit_protocol` | Target provider protocol passed to the host execution path. Default `openai`. |
| `model` | Model identifier. Default `gpt-5.5`; availability depends on the configured model registry and auth records. |
| `stream` | Boolean, default `false`. `stream=true` uses `host.model.execute_stream`. |
| `mode` | Optional shorthand for `stream`: accepts `stream`, `streaming`, `non-stream`, `non_stream`, `nonstream`, or `sync`. |
| `prompt` | Text used to build the default OpenAI-compatible request body. |
| `body` | Optional JSON string in the URL query; replaces the generated body. |
| `alt` | Optional alternate route or mode suffix passed through the host model request. |
| `implicit_close` | Streaming-only boolean, default `false`. |

The generated default body is OpenAI-compatible:

```json
{
  "model": "gpt-5.5",
  "stream": false,
  "messages": [
    {
      "role": "user",
      "content": "Summarize host model callbacks in one short sentence."
    }
  ]
}
```

A URL-encoded `body` query value can provide the raw OpenAI-compatible request instead:

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?body=%7B%22model%22%3A%22gpt-5.5%22%2C%22stream%22%3Afalse%2C%22messages%22%3A%5B%7B%22role%22%3A%22user%22%2C%22content%22%3A%22Say%20hello%20in%20one%20sentence%22%7D%5D%7D
```

## Stream close semantics

By default, streaming mode closes the host-owned stream with `host.model.stream_close` in a deferred call. This is the pattern normal plugins should follow because it releases stream resources as soon as the plugin finishes reading.

With `implicit_close=true`, the plugin skips the explicit close call. The host injects `host_callback_id` into the `management.handle` request, and the plugin forwards that ID to `host.model.execute_stream`, so the host closes the stream when the `management.handle` RPC callback scope returns. This mode exists to demonstrate host cleanup behavior; plugin code that opens streams should close them explicitly.

## Recursion guard

The plugin forwards the `host_callback_id` it received from `management.handle` on every `host.model.execute` and `host.model.execute_stream` call. The host uses that ID to identify the plugin that initiated the callback and skips that plugin's request, response, and stream interceptors for the nested execution. The nested request therefore cannot recurse into this plugin, though other enabled plugins can still intercept it.

## Usage reporting

The callback runs through the same model executor path as normal proxied requests, so usage collection, request accounting, and billing metadata are handled by that shared path. The callback layer adds no usage record of its own.

## Error handling and troubleshooting

The page displays the model status, response headers, body, stream chunks, close mode, and any callback error returned by the host envelope.

- `host model executor is unavailable`: the host model executor path is not initialized for this plugin callback context.
- `unsupported model` or provider-specific routing errors: the `model` value is not routable under the current CPA model/auth configuration.
- `host.model.execute requires stream=false` / `host.model.execute_stream requires stream=true`: the `stream` flag did not match the callback used.
- Empty or partial stream output: check the page error section and host logs; upstream stream errors arrive through `host.model.stream_read`.
