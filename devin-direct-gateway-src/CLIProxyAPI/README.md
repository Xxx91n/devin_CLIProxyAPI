# devin-direct-gateway

[English](README.md) | [简体中文](README_CN.md) | [日本語](README_JA.md)

A fork of [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) that exposes Cognition's Devin through standard OpenAI, Anthropic, and Gemini API endpoints. The upstream providers are all still here; Devin support is what this fork adds and maintains.

## What it does

- One local server serves OpenAI (`/v1/chat/completions`, `/v1/responses`), Anthropic (`/v1/messages`), and Gemini (`/v1beta`) protocols, so existing clients and SDKs work without code changes.
- Multiple accounts per provider: OAuth login flags or API keys in config, with requests balanced across the pool (round-robin by default; `routing.strategy` also supports weighted-round-robin and fill-first).
- Protocol translation between the served formats, including streaming (SSE) responses and a WebSocket route at `/v1/ws`.
- Devin specifics: OAuth PKCE login, Connect-RPC translation to Cognition's backend, and session affinity so a multi-turn conversation stays on one credential. Protocol notes live in [CONTEXT.md](CONTEXT.md).

## Providers

| Provider | Login flag | Credentials |
|---|---|---|
| Devin (Cognition) | `-devin-login` | OAuth, saved under `auth-dir` |
| Codex (OpenAI) | `-codex-login`, `-codex-device-login` | OAuth or `codex-api-key` |
| Claude | `-claude-login` | OAuth or `claude-api-key` |
| Antigravity (Google Gemini) | `-antigravity-login` | OAuth |
| Kimi | `-kimi-login` | OAuth |
| xAI (Grok) | `-xai-login` | OAuth or `xai-api-key` |
| Meta (Muse Code) | `-meta-login` | OAuth or `meta-api-key` |
| Gemini / Vertex | `-vertex-import` | `gemini-api-key` / `vertex-api-key` |
| Any OpenAI-compatible upstream | — | `openai-compatibility` entries (e.g. OpenRouter) |

## Requirements

- Go 1.26 or newer (per `go.mod`), or Docker
- A `config.yaml` based on [config.example.yaml](config.example.yaml)

## Build and run

```bash
go build -o cli-proxy-api ./cmd/server
./cli-proxy-api -config config.yaml
```

Run the tests with `go test ./...`.

Log in to a provider (repeat the flag per account you want in the pool):

```bash
./cli-proxy-api -config config.yaml -devin-login
```

The server listens on port 8317 by default (`port` in the config).

## Configuration

Key settings in `config.yaml` (config.example.yaml documents the full set):

| Key | Purpose | Default |
|---|---|---|
| `port` | Listen port | `8317` |
| `api-keys` | Keys that clients must send | — |
| `auth-dir` | Where OAuth credentials are stored | `~/.cli-proxy-api` |
| `remote-management.secret-key` | Enables the Management API when set; empty disables it | empty |
| `routing.session-affinity` | Bind a session to one credential for the affinity TTL | `false` |
| `proxy-url` | Upstream proxy (socks5/http/https) | empty |

## Limitations

- Starting the server while `api-keys` still holds the example values from config.example.yaml disables the proxy API endpoints until the keys are replaced.
- Devin's OAuth redirect must target `http://127.0.0.1:<port>/callback`. On a remote server the browser cannot reach that loopback page, so login there means copying the final callback URL into the management panel (docs/management-devin-oauth.md).

## API endpoints

| Path | Format |
|---|---|
| `GET /healthz` | Health check |
| `GET /v1/models` | OpenAI or Anthropic model list (negotiated by request headers) |
| `POST /v1/chat/completions` | OpenAI Chat Completions |
| `POST /v1/completions` | OpenAI Completions |
| `POST /v1/responses` | OpenAI Responses |
| `POST /v1/messages` | Anthropic Messages |
| `POST /v1/messages/count_tokens` | Anthropic token counting |
| `POST /v1beta/models/*action` | Gemini generateContent and related calls |
| `POST /v1beta/interactions` | Gemini Interactions |
| `GET /v1/ws` | WebSocket |

## Devin notes

- `./cli-proxy-api -config config.yaml -devin-login` runs the OAuth PKCE flow and saves a `devin-*.json` credential under `auth-dir`.
- OAuth callbacks arrive at `/callback` (also served as `/devin/callback`).
- For browser-based login through the management panel, see [docs/management-devin-oauth.md](docs/management-devin-oauth.md).
- The upstream protocol (Connect-RPC to `server.codeium.com`, trailer errors, session ID handling) is documented in [CONTEXT.md](CONTEXT.md).
- Optional `devin.sensitive-words` config obfuscates listed words in system instructions and prompts.

## Docker

`docker-compose.yml` runs the published image with port 8317 mapped and mounts for `config.yaml`, `auths/`, `logs/`, and `plugins/`:

```bash
docker compose up -d
```

To build the image from source instead, run `./docker-build.sh` and choose option 2.

## Management API and panel

Set `remote-management.secret-key` to enable the `/v0/management/*` routes. With a key set, the bundled web panel is served at `/management.html`; without one, both are disabled. The panel asset downloads from the Cli-Proxy-API-Management-Center GitHub repository on first use.

## SDK

The Go SDK under `sdk/` embeds the proxy in other programs:

- Usage: [docs/sdk-usage.md](docs/sdk-usage.md)
- Executors and translators: [docs/sdk-advanced.md](docs/sdk-advanced.md)
- Access control: [docs/sdk-access.md](docs/sdk-access.md)
- Credential loading: [docs/sdk-watcher.md](docs/sdk-watcher.md)
- Custom provider example: `examples/custom-provider`

## Upstream and credits

This repository is a fork of [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) by Router-For.ME. The management panel comes from [Cli-Proxy-API-Management-Center](https://github.com/router-for-me/Cli-Proxy-API-Management-Center). The sponsor and ecosystem project lists from the upstream README are not reproduced here. Upstream documentation: [help.router-for.me](https://help.router-for.me/).

## License

MIT License; see [LICENSE](LICENSE).
