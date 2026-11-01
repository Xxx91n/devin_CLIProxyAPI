# Claude Code web search router (ModelRouter example)

[English](README.md) | [简体中文](README_CN.md)

A Go plugin for CLIProxyAPI that intercepts Claude Code built-in `web_search` requests and routes them to a search backend: the host's built-in antigravity, codex, or xAI providers through `host.model.*` callbacks, or Tavily directly from the plugin.

## What it detects

A request matches when:

- Inbound protocol is `claude` or `anthropic`.
- `tools[]` contains a tool with `type` `web_search_20250305` or `web_search_20260209`.
- The request looks like Claude Code (system text containing "web search tool use" or "performing a web search", or user text starting with `Perform a web search for the query:`), or its tools list holds only web search tools.

With `require_web_search_only: true` (the default), the tools list must contain only web search tools; mixed tool lists are declined.

## Routes

| `route` value | Behavior |
| --- | --- |
| `fallback` (default) | The plugin executor runs antigravity → codex → xai → tavily. Built-in backends execute through `host.model.*` callbacks; Tavily runs in the plugin. On HTTP 429/503/502 the next backend runs in the same request. Backends that hit those statuses are deprioritized on later requests by an in-memory penalty, with no extra config. |
| `antigravity_google` / `codex_web_search` / `xai_web_search` / `tavily` | Runs only that backend through the same executor path, with the same 429/503/502 handling and penalty recording. |
| `default_provider` | Returns a provider target to the host, which routes to its built-in provider with the optional `default_provider_model`. Not orchestrated by the plugin executor. |

The fallback route handles a request only when at least one backend is runnable: the provider is present in `AvailableProviders` where required, an antigravity model resolves, or `tavily_api_keys` is set.

### xAI notes

- Model: `grok-4.3` when `xai_model` is empty. The client's Claude model name is never forwarded to xAI or Codex.
- Request shape: Responses API `input` with `tools[]` type `web_search`, and optional `filters.allowed_domains` / `filters.excluded_domains` (per xAI docs: the two fields are mutually exclusive and cannot be set in the same request).
- Claude mapping: `internal/translator/codex/claude` copies Claude `allowed_domains` to `filters.allowed_domains`; Claude `blocked_domains` is not forwarded, and mapping it to `filters.excluded_domains` is not yet supported.
- The host's xAI executor drops the unsupported `external_web_access` tool field and posts to `/responses`.
- Citations and server tool metadata come back through the OpenAI Responses SSE path and are converted to Claude `server_tool_use` / `web_search_tool_result` blocks where the response translator supports them.

## Configuration

Config lives under `plugins.configs.<plugin ID>`, where the plugin ID equals the library file name without its extension (for example `claude-web-search-router.dylib` maps to `plugins.configs.claude-web-search-router`). Plugins load only when both `plugins.enabled` and the per-plugin `enabled` are true.

### Recommended: fallback chain (default)

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

Omit `route` for the same `fallback` default.

### Minimal fallback (Tavily as the last step)

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

### Single backends

Each block below replaces the `route` line in the wrapper above; the rest of the wrapper stays the same.

- Antigravity only: `route: antigravity_google`, `antigravity_model: "gemini-3.1-flash-lite"`
- Codex only: `route: codex_web_search`, `codex_model: "gpt-5.4-mini"`
- xAI only: `route: xai_web_search`, `xai_model: "grok-4.3"`
- Tavily only: `route: tavily` plus `tavily_api_keys`
- Built-in provider: `route: default_provider`, `default_provider: claude`, optional `default_provider_model`

### Disable or relax detection

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

### Config fields

| Field | Description |
| --- | --- |
| `enabled` | `false` declines every web_search match (`Handled: false`). |
| `priority` | Plugin order for ModelRouter; higher runs first. |
| `route` | `fallback` (default), `antigravity_google`, `codex_web_search`, `xai_web_search`, `tavily`, `default_provider`. |
| `antigravity_model` | Antigravity execution model. Empty: registry lookup for the requested model, then the first model with `SupportsWebSearch`. |
| `codex_model` | Codex model. Empty: `gpt-5.4-mini`. |
| `xai_model` | xAI model. Empty: `grok-4.3`. |
| `default_provider` / `default_provider_model` | Provider key and optional execution model for `route=default_provider`. |
| `tavily_api_keys` | Required for `route=tavily`; used as the last fallback step when present. Keys rotate round-robin. |
| `require_web_search_only` | `true` (default) requires the tools list to hold only web search tools. |

## Build and install

From the repo root:

```bash
# macOS
make -C examples/plugin bin/claude-web-search-router-go.dylib
# Linux
make -C examples/plugin bin/claude-web-search-router-go.so
# Windows
make -C examples/plugin bin/claude-web-search-router-go.dll
```

Copy the artifact into the plugins directory under the plugin's registered name:

```bash
cp examples/plugin/bin/claude-web-search-router-go.dylib plugins/claude-web-search-router.dylib
```

On Linux or Windows use the `.so` or `.dll` extension in both commands. The config key must equal the copied file name without the extension.
