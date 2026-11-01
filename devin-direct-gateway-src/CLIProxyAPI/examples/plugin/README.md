# CLIProxyAPI plugin examples

[English](README.md) | [简体中文](README_CN.md)

This directory holds the example plugins for the CLIProxyAPI native plugin system. A plugin is a dynamic library the host loads from the configured plugin directory. The host calls the library's `cliproxy_plugin_init` export, and after that the two sides exchange JSON envelopes through a fixed C function table (`internal/pluginhost/loader_unix.go`, `internal/pluginhost/loader_windows.go`). The current ABI version is 1 and the RPC schema version is 6, both defined in `sdk/pluginabi/types.go`.

Plugin loading needs cgo on Unix platforms (Linux, macOS, FreeBSD) and works through `LoadLibrary` on Windows. A binary built without cgo on Unix fails with "standard dynamic library plugin loading requires cgo on this platform" (`internal/pluginhost/loader_unsupported.go`).

## Examples

| Directory | Languages | What it shows |
|---|---|---|
| [`simple`](simple/README.md) | Go, C, Rust | Full mixed-capability skeleton; the reference for the current ABI. |
| [`model`](model/go/main.go) | Go, C, Rust | Model capability only: static models and auth-bound models. |
| [`auth`](auth/go/main.go) | Go, C, Rust | Auth provider capability only: parse, login, poll, refresh. |
| [`frontend-auth`](frontend-auth/go/main.go) | Go, C, Rust | Frontend auth provider capability only. |
| [`frontend-auth-exclusive`](frontend-auth-exclusive/README.md) | Go only | Frontend auth provider that becomes the only request authentication provider when selected. |
| [`executor`](executor/go/main.go) | Go, C, Rust | Executor capability only: execute, streaming, count tokens, HTTP requests. |
| [`protocol-format`](protocol-format/go/main.go) | Go, C, Rust | Minimal executor focused on input and output format declarations. |
| [`request-translator`](request-translator/go/main.go) | Go, C, Rust | Request translation capability only. |
| [`request-normalizer`](request-normalizer/go/main.go) | Go, C, Rust | Request normalization capability only. |
| [`response-translator`](response-translator/go/main.go) | Go, C, Rust | Response translation capability only. |
| [`response-normalizer`](response-normalizer/go/main.go) | Go, C, Rust | Response normalization capability only. |
| [`thinking`](thinking/go/main.go) | Go, C, Rust | Thinking applier capability only. |
| [`usage`](usage/go/main.go) | Go, C, Rust | Usage observer capability only. |
| [`cli`](cli/go/main.go) | Go, C, Rust | Command-line capability only. |
| [`management-api`](management-api/go/main.go) | Go, C, Rust | Management API and resource capability only. |
| [`host-callback`](host-callback/go/main.go) | Go, C, Rust | Minimal Management API route that triggers host HTTP and log callbacks. |
| [`codex-service-tier`](codex-service-tier/README.md) | Go only | Request normalizer that sets Codex `gpt-5.5` requests to the priority service tier when `fast` is `true`. |
| [`request-lifecycle`](request-lifecycle/README.md) | Go only | Request admission with concurrency control, active HTTP termination, and terminal callbacks. |
| [`scheduler`](scheduler/README.md) | Go only | Scheduler that selects a configured auth ID, delegates to a built-in scheduler, or denies picks. |
| [`claude-web-search-router`](claude-web-search-router/README.md) | Go only | ModelRouter plus executor for Claude Code built-in `web_search` (antigravity / codex / xai / Tavily). |
| [`host-callback-auth-files`](host-callback-auth-files/README.md) | Go only | Management API resource that calls the host auth file callbacks. |
| [`host-model-callback`](host-model-callback/README.md) | Go only | Management API resource that runs model execution through the host model callbacks. |

`scripts/generate_examples.py` generates the single-capability examples from a capability table, so the C, Go, and Rust implementations of one capability stay in sync.

## Plugin configuration

The host reads these keys from the top-level `plugins` section of the CPA config (`internal/config/config_types.go:40-53`):

- `enabled`: master switch. When `false`, the host loads no plugins at all (`internal/pluginhost/config.go:38-41`).
- `dir`: plugin discovery directory, default `plugins`.
- `store-sources`, `store-auth`: third-party plugin store registries and their auth rules.
- `auth-revision`: bumped when Home-managed plugin credentials change.
- `configs`: per-plugin settings keyed by plugin ID.

Each entry under `configs` has two host-owned fields, `enabled` and `priority`; an entry without `enabled: true` stays disabled. The remaining YAML subtree is passed to the plugin as its config.

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-service-tier:
      enabled: true
      priority: 1
      fast: true
```

## Build

Prerequisites: Go, CMake (C examples), and Cargo (Rust examples).

```bash
make -C examples/plugin list
make -C examples/plugin build
```

Artifacts land in `examples/plugin/bin` as `<example>-<go|c|rust>.<ext>`. The extension follows the OS: `.dylib` on macOS, `.so` on Linux, `.dll` on Windows (see the pattern rules in `examples/plugin/Makefile`).

The Makefile builds the examples listed in its `EXAMPLES` variable. Four Go-only examples are absent from that list — `codex-service-tier`, `request-lifecycle`, `scheduler`, and `frontend-auth-exclusive` — and build directly with `go build -buildmode=c-shared` as described in their own READMEs.

## Codex Service Tier

`codex-service-tier` declares the request normalization capability. When `fast` is `true`, it sets `service_tier` to `priority` for requests where `req.ToFormat` is `codex` and `req.Model` is `gpt-5.5`.

```yaml
plugins:
  configs:
    codex-service-tier:
      enabled: true
      priority: 1
      fast: false
```

## Request Lifecycle

`request-lifecycle` combines `request_interceptor` with `request_lifecycle_plugin`. It acquires a concurrency slot before auth selection, can return a custom `403` or `429` response without contacting an upstream model, and releases admitted slots from `request.complete` on success, failure, rejection, or cancellation.

```yaml
plugins:
  configs:
    request-lifecycle:
      enabled: true
      priority: 100
      max_concurrency: 2
      reject_keyword: "blocked"
```

See `request-lifecycle/README.md` for build instructions and lifecycle semantics.

## Host Auth Files Callback

`host-callback-auth-files` declares the Management API capability and exposes a browser resource named `Host Auth Files`. The resource demonstrates `host.auth.list`, `host.auth.get` (physical JSON file), `host.auth.get_runtime`, and `host.auth.save`.

```yaml
plugins:
  configs:
    host-callback-auth-files:
      enabled: true
      priority: 1
```

See `host-callback-auth-files/README.md` for URL examples.

## Host Model Callback

`host-model-callback` declares the Management API capability and exposes a browser resource named `Host Model Callback`. The resource calls `host.model.execute` for non-streaming requests and `host.model.execute_stream` plus `host.model.stream_read` for streaming requests. It demonstrates explicit stream close with `host.model.stream_close` and an `implicit_close=true` option for RPC-scope host cleanup.

When the resource forwards its `host_callback_id`, CPA identifies the plugin that initiated the host model callback and skips that same plugin's interceptors for the nested execution. This makes host model callbacks non-recursive for the caller while allowing other plugins to intercept the nested request.

```yaml
plugins:
  configs:
    host-model-callback:
      enabled: true
      priority: 1
```

The default example model is `gpt-5.5`, but the request succeeds only when the current CPA model and auth configuration can route that model.

## Scheduler

`scheduler` declares the scheduler capability. It can select a configured auth ID from the candidate list, delegate to the built-in `fill-first` or `round-robin` scheduler, or reject picks when `deny` is `true`.

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

`auth_id` selects a matching candidate when `delegate` is empty. `delegate` accepts `""`, `fill-first`, or `round-robin`; other non-empty values leave the pick unhandled. `deny` returns a scheduler error.

## Plugin Executor Error Handling and HTTP Status

When a plugin executor encounters an upstream failure (such as `401 Unauthorized` for invalid credentials, `403 Forbidden` for model permission/quota limits, or `429 Too Many Requests` for rate limits), it should report the HTTP status code in the error envelope:

- In the JSON RPC error envelope, set the `http_status` field inside the `error` object (`pluginabi.Error.HTTPStatus`).
- If `http_status` is omitted or `0`, CPA defaults to returning HTTP `500 Internal Server Error` (`server_error` / `internal_server_error`), which clients typically treat as a temporary gateway outage and retry with backoff.
- When `http_status` is set, CPA maps the status into client-visible error responses:
  - `401` -> HTTP 401 with `type: "authentication_error"`, `code: "invalid_api_key"`
  - `403` -> HTTP 403 with `type: "permission_error"`, `code: "upstream_permission_denied"`; the code becomes `insufficient_quota` when the error message mentions quota, credit, ACU, or exhaustion
  - `429` -> HTTP 429 with `type: "rate_limit_error"`, `code: "rate_limit_exceeded"`
  - `404` -> HTTP 404 with `type: "invalid_request_error"`, `code: "model_not_found"`
  - `>=500` -> HTTP 5xx with `type: "server_error"`, `code: "internal_server_error"`
- **Important**: Both non-streaming (`executor.execute`) and streaming (`executor.execute_stream`) call sites must include `http_status` so failures are classified consistently.
- Because native dynamic library plugins communicate across the C ABI via serialized JSON buffers, the status code must be encoded in the serialized JSON envelope (e.g. using `pluginabi.NewErrorEnvelope` or a custom envelope struct with an `http_status` field). Returning an unmarshaled Go error does not traverse the C ABI boundary.

Go plugins can import `github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi` and use `pluginabi.NewErrorEnvelope(code, message, httpStatus)`:

```go
// Recommended: construct an error envelope directly using sdk/pluginabi
rawEnvelope, errMarshal := pluginabi.NewErrorEnvelope("insufficient_quota", "plan limit reached", http.StatusForbidden)
```

Alternatively, plugins defining a custom envelope struct can declare an `HTTPStatus` field (`json:"http_status,omitempty"`):

```go
type envelopeError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

func errorEnvelope(code, message string, httpStatus ...int) []byte {
	status := 0
	if len(httpStatus) > 0 {
		status = httpStatus[0]
	}
	raw, _ := json.Marshal(envelope{
		OK: false,
		Error: &envelopeError{
			Code:       code,
			Message:    message,
			HTTPStatus: status,
		},
	})
	return raw
}
```

## Notes

`protocol-format` uses a minimal executor because format declarations belong to executor capabilities.

`host-callback` uses a minimal plugin resource because host callbacks are invoked from plugin methods and are not standalone capabilities.

Menu resources returned by `management.register` through the `resources` field are exposed by CPA under `/v0/resource/plugins/<pluginID>/...`. Authenticated plugin Management API routes remain under `/v0/management/...`.
