# CLIProxyAPI 插件示例

[English](README.md) | [简体中文](README_CN.md)

本目录是 CLIProxyAPI 原生插件系统的示例集合。插件是一个动态库，宿主从配置的插件目录加载它，调用其 `cliproxy_plugin_init` 导出函数，之后双方通过固定的 C 函数表交换 JSON 信封（`internal/pluginhost/loader_unix.go`、`internal/pluginhost/loader_windows.go`）。当前 ABI 版本为 1，RPC schema 版本为 6，均定义在 `sdk/pluginabi/types.go` 中。

Unix 平台（Linux、macOS、FreeBSD）加载插件需要 cgo，Windows 通过 `LoadLibrary` 实现。Unix 上未启用 cgo 的构建会报错 "standard dynamic library plugin loading requires cgo on this platform"（`internal/pluginhost/loader_unsupported.go`）。

## 示例列表

| 目录 | 语言 | 演示内容 |
|---|---|---|
| [`simple`](simple/README.md) | Go、C、Rust | 完整的多能力骨架，是当前 ABI 的参考实现。 |
| [`model`](model/go/main.go) | Go、C、Rust | 只演示模型能力：静态模型与按认证发现模型。 |
| [`auth`](auth/go/main.go) | Go、C、Rust | 只演示认证提供方能力：解析、登录、轮询、刷新。 |
| [`frontend-auth`](frontend-auth/go/main.go) | Go、C、Rust | 只演示前端认证提供方能力。 |
| [`frontend-auth-exclusive`](frontend-auth-exclusive/README.md) | 仅 Go | 演示被选中后成为唯一请求认证方式的前端认证提供方。 |
| [`executor`](executor/go/main.go) | Go、C、Rust | 只演示执行器能力：普通执行、流式执行、计数、HTTP 请求。 |
| [`protocol-format`](protocol-format/go/main.go) | Go、C、Rust | 使用最小执行器重点演示输入和输出格式声明。 |
| [`request-translator`](request-translator/go/main.go) | Go、C、Rust | 只演示请求转换能力。 |
| [`request-normalizer`](request-normalizer/go/main.go) | Go、C、Rust | 只演示请求规整能力。 |
| [`response-translator`](response-translator/go/main.go) | Go、C、Rust | 只演示响应转换能力。 |
| [`response-normalizer`](response-normalizer/go/main.go) | Go、C、Rust | 只演示响应规整能力。 |
| [`thinking`](thinking/go/main.go) | Go、C、Rust | 只演示 Thinking 处理能力。 |
| [`usage`](usage/go/main.go) | Go、C、Rust | 只演示 Usage 观察能力。 |
| [`cli`](cli/go/main.go) | Go、C、Rust | 只演示命令行扩展能力。 |
| [`management-api`](management-api/go/main.go) | Go、C、Rust | 只演示 Management API 和资源扩展能力。 |
| [`host-callback`](host-callback/go/main.go) | Go、C、Rust | 用最小 Management API 路由触发宿主 HTTP 和日志回调。 |
| [`codex-service-tier`](codex-service-tier/README.md) | 仅 Go | 请求规整插件，启用 `fast` 后将 Codex `gpt-5.5` 请求设置为 priority service tier。 |
| [`request-lifecycle`](request-lifecycle/README.md) | 仅 Go | 请求生命周期插件，演示并发控制、主动终止 HTTP 请求和终态回调。 |
| [`scheduler`](scheduler/README.md) | 仅 Go | 调度插件，可选择指定 auth ID、委托内置调度器或拒绝调度。 |
| [`claude-web-search-router`](claude-web-search-router/README.md) | 仅 Go | ModelRouter 加执行器，接管 Claude Code 内置 `web_search`（antigravity / codex / xai / Tavily）。 |
| [`host-callback-auth-files`](host-callback-auth-files/README.md) | 仅 Go | Management API 资源，演示调用 host 凭证文件回调。 |
| [`host-model-callback`](host-model-callback/README.md) | 仅 Go | Management API 资源，演示通过宿主模型回调执行模型请求。 |

`scripts/generate_examples.py` 依据能力表生成单能力示例，使同一能力的 C、Go、Rust 三份实现保持一致。

## 插件配置

宿主从 CPA 配置顶层的 `plugins` 段读取以下键（`internal/config/config_types.go:40-53`）：

- `enabled`：总开关。为 `false` 时宿主完全不加载插件（`internal/pluginhost/config.go:38-41`）。
- `dir`：插件发现目录，默认 `plugins`。
- `store-sources`、`store-auth`：第三方插件商店注册源及其认证规则。
- `auth-revision`：Home 托管的插件凭证变化时递增。
- `configs`：以插件 ID 为键的每个插件设置。

`configs` 下每个条目有两个宿主持有的字段：`enabled` 和 `priority`；条目未写 `enabled: true` 时保持禁用。其余 YAML 子树会作为插件自身的配置传给插件。

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

## 构建

前置条件（prerequisites）：Go，CMake（C 示例），Cargo（Rust 示例）。

```bash
make -C examples/plugin list
make -C examples/plugin build
```

构建产物写入 `examples/plugin/bin`，命名格式为 `<example>-<go|c|rust>.<ext>`。扩展名跟随操作系统：macOS 为 `.dylib`，Linux 为 `.so`，Windows 为 `.dll`（见 `examples/plugin/Makefile` 的模式规则）。

Makefile 只构建其 `EXAMPLES` 变量中列出的示例。四个仅 Go 的示例不在该列表中——`codex-service-tier`、`request-lifecycle`、`scheduler`、`frontend-auth-exclusive`——需按各自 README 用 `go build -buildmode=c-shared` 直接构建。

## Codex Service Tier

`codex-service-tier` 声明请求规整能力。当 `fast` 为 `true` 时，如果 `req.ToFormat` 为 `codex` 且 `req.Model` 为 `gpt-5.5`，它会将 `service_tier` 设置为 `priority`。

```yaml
plugins:
  configs:
    codex-service-tier:
      enabled: true
      priority: 1
      fast: false
```

## 请求生命周期

`request-lifecycle` 同时声明 `request_interceptor` 和 `request_lifecycle_plugin`。它会在认证选择前占用并发槽位，可以直接返回自定义 `403` 或 `429` 响应而不请求上游模型，并在成功、失败、拒绝或取消时通过 `request.complete` 释放已接入请求的槽位。

```yaml
plugins:
  configs:
    request-lifecycle:
      enabled: true
      priority: 100
      max_concurrency: 2
      reject_keyword: "blocked"
```

构建方式和生命周期语义详见 `request-lifecycle/README.md`。

## Host Auth Files 回调

`host-callback-auth-files` 声明 Management API 能力，并暴露名为 `Host Auth Files` 的浏览器资源，演示 `host.auth.list`、`host.auth.get`（物理 JSON 文件）、`host.auth.get_runtime` 与 `host.auth.save`。

```yaml
plugins:
  configs:
    host-callback-auth-files:
      enabled: true
      priority: 1
```

详见 `host-callback-auth-files/README.md`。

## Host Model Callback

`host-model-callback` 声明 Management API 能力，并暴露名为 `Host Model Callback` 的浏览器资源。该资源在非流式请求中调用 `host.model.execute`，在流式请求中调用 `host.model.execute_stream` 和 `host.model.stream_read`。它演示了通过 `host.model.stream_close` 显式关闭流，也提供 `implicit_close=true` 用于演示 RPC 作用域结束时的宿主隐式清理。

当该资源转发自身收到的 `host_callback_id` 时，CPA 会识别发起宿主模型回调的插件，并在嵌套模型执行中跳过同一个插件的拦截器。因此宿主模型回调不会递归调用发起插件自身，但其他已启用插件仍可拦截这次嵌套请求。

```yaml
plugins:
  configs:
    host-model-callback:
      enabled: true
      priority: 1
```

默认示例模型是 `gpt-5.5`，但请求能否成功取决于当前 CPA 模型和认证配置是否可以路由该模型。

## Scheduler

`scheduler` 声明调度能力。它可以从候选列表中选择配置的 auth ID，委托内置的 `fill-first` 或 `round-robin` 调度器，或在 `deny` 为 `true` 时拒绝调度。

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

`auth_id` 会在 `delegate` 为空时选择匹配候选。`delegate` 支持 `""`、`fill-first` 和 `round-robin`；其他非空值会让本插件不处理本次调度。`deny` 会返回调度错误。

## 插件执行器错误处理与 HTTP 状态码

当插件执行器（Executor）遇到上游调用失败（如因凭据无效返回 `401 Unauthorized`、因模型权限或配额限制返回 `403 Forbidden`，或因限流返回 `429 Too Many Requests`）时，应在错误信封中设置 HTTP 状态码：

- 在 JSON RPC 错误信封中，设置 `error` 对象内的 `http_status` 字段（对应 `pluginabi.Error.HTTPStatus`）。
- 若省略 `http_status` 或设置为 `0`，CPA 会默认返回 HTTP `500 Internal Server Error`（`server_error` / `internal_server_error`），客户端通常将其视为临时网关故障并退避重试。
- 显式设置 `http_status` 后，CPA 会将状态码映射为客户端可见的错误响应：
  - `401` -> HTTP 401，`type: "authentication_error"`，`code: "invalid_api_key"`
  - `403` -> HTTP 403，`type: "permission_error"`，`code: "upstream_permission_denied"`；当错误消息提到 quota、credit、ACU 或 exhaustion 时，code 变为 `insufficient_quota`
  - `429` -> HTTP 429，`type: "rate_limit_error"`，`code: "rate_limit_exceeded"`
  - `404` -> HTTP 404，`type: "invalid_request_error"`，`code: "model_not_found"`
  - `>=500` -> HTTP 5xx，`type: "server_error"`，`code: "internal_server_error"`
- **注意**：非流式执行（`executor.execute`）和流式执行（`executor.execute_stream`）两处报错路径均需要设置 `http_status`，以保证异常分类行为一致。
- 原生动态库插件通过 C ABI 交换序列化的 JSON 缓冲区通信，因此状态码必须编码到序列化的 JSON 信封中（例如使用 `pluginabi.NewErrorEnvelope` 或自定义带 `http_status` 字段的信封结构体）。Go 语言原生的 error 对象无法跨越 C ABI 边界传递。

Go 语言编写的插件可导入 `github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi` 并直接使用 `pluginabi.NewErrorEnvelope(code, message, httpStatus)`：

```go
// 推荐方式：直接使用 sdk/pluginabi 构造错误信封
rawEnvelope, errMarshal := pluginabi.NewErrorEnvelope("insufficient_quota", "plan limit reached", http.StatusForbidden)
```

也可以在自定义信封结构体中声明 `HTTPStatus` 字段（`json:"http_status,omitempty"`）：

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

## 说明

`protocol-format` 使用最小执行器承载，因为格式声明属于执行器能力。

`host-callback` 使用最小插件资源承载，因为宿主回调只能从插件方法内部发起，不是独立能力。

`management.register` 通过 `resources` 字段返回的菜单资源会由 CPA 暴露在 `/v0/resource/plugins/<pluginID>/...` 下。需要认证的插件自有 Management API 路由仍保留在 `/v0/management/...` 下。
