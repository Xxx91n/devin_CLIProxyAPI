# 宿主模型回调插件

[English](README.md) | [简体中文](README_CN.md)

一个仅 Go 实现的示例插件：其 Management API 页面通过宿主模型回调执行模型请求，插件自身不发出任何出站 HTTP 请求。

## 用途与范围

插件在 `/status` 注册了一个名为 `Host Model Callback` 的 Management API 资源，暴露路径为：

```text
/v0/resource/plugins/host-model-callback/status
```

页面由查询参数驱动：读取 URL 查询参数，构造 OpenAI 兼容的聊天请求，然后调用：

- `host.model.execute`：非流式执行。
- `host.model.execute_stream`、`host.model.stream_read`、`host.model.stream_close`：流式执行。

本示例只演示宿主模型回调。它不注册 executor、translator、normalizer、auth provider、scheduler 能力，也不包含出站 HTTP 客户端。

## 构建

在本目录下执行：

```bash
cd go
go build -buildmode=c-shared -o host-model-callback.dylib .
rm -f host-model-callback.h
# 复制到配置的 plugins.dir。下面的路径对应服务端从 CLIProxyAPI 根目录启动时的
# 默认 "plugins" 目录。
cp host-model-callback.dylib ../../../../plugins/
```

按目标系统使用对应扩展名：macOS 用 `.dylib`，Linux 用 `.so`，Windows 用 `.dll`。

## 配置

把动态库放入所配置的插件目录（`plugins.dir`，默认 `plugins`），文件基名需与插件 ID 一致：`plugins/host-model-callback.dylib` 对应 `plugins.configs.host-model-callback`。

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    host-model-callback:
      enabled: true
      priority: 1
```

本插件没有自己的配置字段。

## 资源 URL 示例

非流式请求，全部使用默认值：

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status
```

非流式请求，显式指定协议与提示词：

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?entry_protocol=openai&exit_protocol=openai&model=gpt-5.5&prompt=Say%20hello%20in%20one%20sentence
```

流式请求，显式关闭：

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?stream=true&model=gpt-5.5&prompt=Write%20three%20short%20tokens
```

流式请求，依赖 RPC 作用域的隐式关闭：

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?stream=true&implicit_close=true
```

默认模型 ID 是 `gpt-5.5`，与相邻的 `codex-service-tier` 示例保持一致。它只是示例标识；请求能否成功取决于你的 CPA 配置能否路由该模型。

## 参数

| 参数 | 说明 |
| --- | --- |
| `entry_protocol` | 传给宿主执行路径的入站客户端协议。默认 `openai`。 |
| `exit_protocol` | 传给宿主执行路径的目标提供商协议。默认 `openai`。 |
| `model` | 模型标识。默认 `gpt-5.5`；可用性取决于已配置的模型注册表与认证记录。 |
| `stream` | 布尔值，默认 `false`。`stream=true` 使用 `host.model.execute_stream`。 |
| `mode` | `stream` 的另一种写法：接受 `stream`、`streaming`、`non-stream`、`non_stream`、`nonstream`、`sync`。 |
| `prompt` | 用于构造默认 OpenAI 兼容请求体的文本。 |
| `body` | 可选，URL 查询中的 JSON 字符串；提供时替换生成的请求体。 |
| `alt` | 可选，传给宿主模型请求的备用路由或模式后缀。 |
| `implicit_close` | 仅流式生效的布尔值，默认 `false`。 |

生成的默认请求体是 OpenAI 兼容的：

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

也可以用 URL 编码的 `body` 查询值直接提供原始 OpenAI 兼容请求：

```text
http://localhost:8080/v0/resource/plugins/host-model-callback/status?body=%7B%22model%22%3A%22gpt-5.5%22%2C%22stream%22%3Afalse%2C%22messages%22%3A%5B%7B%22role%22%3A%22user%22%2C%22content%22%3A%22Say%20hello%20in%20one%20sentence%22%7D%5D%7D
```

## 流关闭语义

默认情况下，流式模式通过延迟调用 `host.model.stream_close` 显式关闭宿主持有的流。这是普通插件应遵循的模式：插件读完数据后立即释放流资源。

设置 `implicit_close=true` 时，插件跳过显式关闭。宿主会在 `management.handle` 请求中注入 `host_callback_id`，插件把这个 ID 转发给 `host.model.execute_stream`，宿主因此在 `management.handle` RPC 回调作用域返回时关闭该流。这种模式只用于演示宿主清理行为；打开流的插件代码应当显式关闭。

## 递归防护

插件在每次调用 `host.model.execute` 和 `host.model.execute_stream` 时都会转发从 `management.handle` 收到的 `host_callback_id`。宿主用这个 ID 识别发起回调的插件，并在嵌套执行中跳过该插件自身的请求、响应和流拦截器。因此嵌套请求不会递归进入本插件，但其他已启用的插件仍可拦截该请求。

## 用量统计

回调走与普通代理请求相同的模型执行器路径，用量采集、请求记账与计费元数据都由这条共享路径处理。回调层自身不会额外生成用量记录。

## 错误处理与排查

页面会显示模型状态、响应头、响应体、流分块、关闭模式，以及宿主信封返回的回调错误。

- `host model executor is unavailable`：当前插件回调上下文中宿主模型执行器路径未初始化。
- `unsupported model` 或提供商路由错误：`model` 值在当前 CPA 模型/认证配置下不可路由。
- `host.model.execute requires stream=false` / `host.model.execute_stream requires stream=true`：`stream` 标志与所调用的回调不匹配。
- 流输出为空或不完整：查看页面的错误区域和宿主日志；上游流错误经 `host.model.stream_read` 返回。
