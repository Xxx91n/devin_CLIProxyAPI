# devin-direct-gateway

[English](README.md) | 简体中文 | [日本語](README_JA.md)

本仓库是 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 的 fork，把 Cognition 的 Devin 接入标准的 OpenAI、Anthropic、Gemini API 端点。上游的提供商全部保留；本 fork 新增并维护的是 Devin 支持。

## 功能

- 一个本地服务同时提供 OpenAI（`/v1/chat/completions`、`/v1/responses`）、Anthropic（`/v1/messages`）和 Gemini（`/v1beta`）协议，现有客户端和 SDK 无需改代码即可接入。
- 每个提供商可登录多个账号：OAuth 登录参数或配置中的 API key，请求在账号池间负载均衡（默认轮询；`routing.strategy` 另支持 weighted-round-robin 和 fill-first）。
- 服务格式之间的协议转换，包括流式（SSE）响应和 `/v1/ws` WebSocket 通道。
- Devin 部分：OAuth PKCE 登录、到 Cognition 后端的 Connect-RPC 转换、会话亲和（同一多轮对话绑定同一凭据）。协议说明见 [CONTEXT.md](CONTEXT.md)。

## 提供商

| 提供商 | 登录参数 | 凭据 |
|---|---|---|
| Devin (Cognition) | `-devin-login` | OAuth，保存在 `auth-dir` 下 |
| Codex (OpenAI) | `-codex-login`、`-codex-device-login` | OAuth 或 `codex-api-key` |
| Claude | `-claude-login` | OAuth 或 `claude-api-key` |
| Antigravity (Google Gemini) | `-antigravity-login` | OAuth |
| Kimi | `-kimi-login` | OAuth |
| xAI (Grok) | `-xai-login` | OAuth 或 `xai-api-key` |
| Meta (Muse Code) | `-meta-login` | OAuth 或 `meta-api-key` |
| Gemini / Vertex | `-vertex-import` | `gemini-api-key` / `vertex-api-key` |
| 任意 OpenAI 兼容上游 | — | `openai-compatibility` 配置项（如 OpenRouter） |

## 环境要求

- Go 1.26 或更高版本（见 `go.mod`），或使用 Docker
- 基于 [config.example.yaml](config.example.yaml) 编写的 `config.yaml`

## 构建与运行

```bash
go build -o cli-proxy-api ./cmd/server
./cli-proxy-api -config config.yaml
```

运行测试：`go test ./...`。

登录一个提供商（每个账号执行一次，逐个加入账号池）：

```bash
./cli-proxy-api -config config.yaml -devin-login
```

服务默认监听 8317 端口（配置项 `port`）。

## 配置

`config.yaml` 中的关键配置（完整列表见 config.example.yaml）：

| 配置项 | 用途 | 默认值 |
|---|---|---|
| `port` | 监听端口 | `8317` |
| `api-keys` | 客户端必须携带的 key | — |
| `auth-dir` | OAuth 凭据保存目录 | `~/.cli-proxy-api` |
| `remote-management.secret-key` | 设置后启用 Management API，留空则禁用 | 空 |
| `routing.session-affinity` | 在亲和 TTL 内把会话绑定到同一凭据 | `false` |
| `proxy-url` | 上游代理（socks5/http/https） | 空 |

## 已知限制 (Known limitations)

- `api-keys` 仍是 config.example.yaml 中的示例值时启动服务，代理 API 端点会被禁用，直到换成真实 key。
- Devin 的 OAuth 重定向必须指向 `http://127.0.0.1:<port>/callback`。远程服务器上浏览器无法访问该回环页面，因此需要把最终回调 URL 粘贴到管理面板完成登录（docs/management-devin-oauth.md）。

## API 端点

| 路径 | 协议 |
|---|---|
| `GET /healthz` | 健康检查 |
| `GET /v1/models` | OpenAI 或 Anthropic 模型列表（按请求头协商） |
| `POST /v1/chat/completions` | OpenAI Chat Completions |
| `POST /v1/completions` | OpenAI Completions |
| `POST /v1/responses` | OpenAI Responses |
| `POST /v1/messages` | Anthropic Messages |
| `POST /v1/messages/count_tokens` | Anthropic token 计数 |
| `POST /v1beta/models/*action` | Gemini generateContent 及相关调用 |
| `POST /v1beta/interactions` | Gemini Interactions |
| `GET /v1/ws` | WebSocket |

## Devin 说明

- `./cli-proxy-api -config config.yaml -devin-login` 运行 OAuth PKCE 流程，并把 `devin-*.json` 凭据保存到 `auth-dir` 下。
- OAuth 回调落在 `/callback`（也以 `/devin/callback` 提供）。
- 通过管理面板在浏览器中登录的流程见 [docs/management-devin-oauth.md](docs/management-devin-oauth.md)。
- 上游协议（到 `server.codeium.com` 的 Connect-RPC、trailer 错误、会话 ID 处理）记录在 [CONTEXT.md](CONTEXT.md)。
- 可选配置 `devin.sensitive-words` 可对系统指令和提示词中的指定词做混淆。

## Docker

`docker-compose.yml` 使用已发布的镜像，映射 8317 端口，并挂载 `config.yaml`、`auths/`、`logs/`、`plugins/`：

```bash
docker compose up -d
```

如需从源码构建镜像，运行 `./docker-build.sh` 并选择选项 2。

## Management API 与管理面板

设置 `remote-management.secret-key` 后启用 `/v0/management/*` 路由。设置密钥后，内置 Web 面板在 `/management.html` 提供；未设置时两者均禁用。面板资源在首次访问时从 Cli-Proxy-API-Management-Center 的 GitHub 仓库下载。

## SDK

`sdk/` 下的 Go SDK 可把代理嵌入其他程序：

- 使用文档：[docs/sdk-usage_CN.md](docs/sdk-usage_CN.md)
- 执行器与转换器：[docs/sdk-advanced_CN.md](docs/sdk-advanced_CN.md)
- 访问控制：[docs/sdk-access_CN.md](docs/sdk-access_CN.md)
- 凭据加载：[docs/sdk-watcher_CN.md](docs/sdk-watcher_CN.md)
- 自定义 Provider 示例：`examples/custom-provider`

## 上游与致谢

本仓库 fork 自 Router-For.ME 的 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)。管理面板来自 [Cli-Proxy-API-Management-Center](https://github.com/router-for-me/Cli-Proxy-API-Management-Center)。上游 README 中的赞助商与生态项目列表在此不再转载。上游文档：[help.router-for.me](https://help.router-for.me/)。

## 许可证

MIT 许可证；详见 [LICENSE](LICENSE)。
