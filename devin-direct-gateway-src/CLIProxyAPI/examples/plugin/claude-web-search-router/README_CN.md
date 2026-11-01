# Claude Code 网页搜索路由（ModelRouter 示例）

[English](README.md) | [简体中文](README_CN.md)

一个 CLIProxyAPI 的 Go 插件示例：拦截 Claude Code 内置的 `web_search` 请求并转发到搜索后端——通过 `host.model.*` 回调使用宿主内置的 antigravity、codex、xAI 提供商，或由插件直接调用 Tavily。

## 识别条件

一个请求会被匹配，当：

- 入站协议为 `claude` 或 `anthropic`。
- `tools[]` 中存在 `type` 为 `web_search_20250305` 或 `web_search_20260209` 的工具。
- 请求看起来来自 Claude Code（system 文本包含 "web search tool use" 或 "performing a web search"，或用户文本以 `Perform a web search for the query:` 开头），或者其工具列表只包含网页搜索工具。

`require_web_search_only: true`（默认值）时，工具列表只能包含网页搜索工具；混合工具列表会被拒绝。

## 路由

| `route` 取值 | 行为 |
| --- | --- |
| `fallback`（默认） | 插件执行器按 antigravity → codex → xai → tavily 顺序运行。内置后端通过 `host.model.*` 回调执行；Tavily 在插件内执行。遇到 HTTP 429/503/502 时，同一请求内尝试下一个后端。频繁命中这些状态码的后端会按内存中的惩罚值在后续请求中被降低优先级，无需额外配置。 |
| `antigravity_google` / `codex_web_search` / `xai_web_search` / `tavily` | 只运行该后端，走同一执行器路径，429/503/502 处理和惩罚记录相同。 |
| `default_provider` | 向宿主返回一个 provider 目标，由宿主路由到其内置提供商，可选 `default_provider_model`。不经插件执行器编排。 |

`fallback` 路由只在至少一个后端可运行时接手请求：所需的提供商出现在 `AvailableProviders` 中、antigravity 模型可解析，或设置了 `tavily_api_keys`。

### xAI 说明

- 模型：`xai_model` 为空时使用 `grok-4.3`。客户端的 Claude 模型名永远不会转发给 xAI 或 Codex。
- 请求形态：Responses API 的 `input`，`tools[]` 类型为 `web_search`，可选 `filters.allowed_domains` / `filters.excluded_domains`（xAI 文档：两个字段互斥，不能同时出现在同一请求中）。
- Claude 映射：`internal/translator/codex/claude` 会把 Claude 的 `allowed_domains` 复制为 `filters.allowed_domains`；Claude 的 `blocked_domains` 暂不支持映射为 `filters.excluded_domains`，也不会原样转发。
- 宿主的 xAI 执行器会丢弃不支持的 `external_web_access` 工具字段，并 POST 到 `/responses`。
- 引用与 server tool 元数据经 OpenAI Responses SSE 返回，在响应翻译器支持范围内转换为 Claude 的 `server_tool_use` / `web_search_tool_result` 块。

## 配置

配置位于 `plugins.configs.<插件 ID>` 之下，插件 ID 等于库文件名去掉扩展名（例如 `claude-web-search-router.dylib` 对应 `plugins.configs.claude-web-search-router`）。只有 `plugins.enabled` 和插件级 `enabled` 同时为 true 时插件才会加载。

### 推荐：fallback 链（默认）

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    claude-web-search-router:
      enabled: true
      priority: 20
      route: fallback
      antigravity_model: "" # empty: registry lookup, then first supports_web_search
      codex_model: "gpt-5.4-mini"
      xai_model: "grok-4.3"
      tavily_api_keys:
        - "tvly-xxxxxxxx"
        # - "tvly-yyyyyyyy" # optional: round-robin
      require_web_search_only: true
```

省略 `route` 即使用同样的 `fallback` 默认值。

### 最小 fallback（Tavily 只作最后一环）

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    claude-web-search-router:
      enabled: true
      priority: 20
      route: fallback
      tavily_api_keys:
        - "tvly-xxxxxxxx"
      require_web_search_only: true
```

### 单后端

下面每种配置只需替换上面模板中的 `route` 一行，其余保持不变。

- 仅 Antigravity：`route: antigravity_google`，`antigravity_model: "gemini-3.1-flash-lite"`
- 仅 Codex：`route: codex_web_search`，`codex_model: "gpt-5.4-mini"`
- 仅 xAI：`route: xai_web_search`，`xai_model: "grok-4.3"`
- 仅 Tavily：`route: tavily`，并配置 `tavily_api_keys`
- 内置提供商：`route: default_provider`，`default_provider: claude`，可选 `default_provider_model`

### 关闭或放宽识别

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    claude-web-search-router:
      enabled: true
      route: fallback
      require_web_search_only: false # accept requests that mix web_search with other tools
      # set enabled: false to decline every web_search match
```

### 配置字段

| 字段 | 说明 |
| --- | --- |
| `enabled` | `false` 时拒绝所有 web_search 匹配（`Handled: false`）。 |
| `priority` | ModelRouter 的插件顺序；数值越大越先运行。 |
| `route` | `fallback`（默认）、`antigravity_google`、`codex_web_search`、`xai_web_search`、`tavily`、`default_provider`。 |
| `antigravity_model` | Antigravity 执行模型。为空时先按请求模型查注册表，再取第一个 `SupportsWebSearch` 的模型。 |
| `codex_model` | Codex 模型。为空时使用 `gpt-5.4-mini`。 |
| `xai_model` | xAI 模型。为空时使用 `grok-4.3`。 |
| `default_provider` / `default_provider_model` | `route=default_provider` 使用的提供商键与可选执行模型。 |
| `tavily_api_keys` | `route=tavily` 时必填；存在时作为 fallback 的最后一环。多个 key 轮流使用。 |
| `require_web_search_only` | `true`（默认）要求工具列表只包含网页搜索工具。 |

## 构建与安装

在仓库根目录执行：

```bash
# macOS
make -C examples/plugin bin/claude-web-search-router-go.dylib
# Linux
make -C examples/plugin bin/claude-web-search-router-go.so
# Windows
make -C examples/plugin bin/claude-web-search-router-go.dll
```

把产物复制到插件目录，并使用插件注册名：

```bash
cp examples/plugin/bin/claude-web-search-router-go.dylib plugins/claude-web-search-router.dylib
```

Linux 或 Windows 上两条命令分别改用 `.so` 或 `.dll` 扩展名。配置键必须等于复制后的文件名去掉扩展名。
