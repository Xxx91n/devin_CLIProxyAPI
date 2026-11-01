# devin_CLIProxyAPI

[English](README.md) | [简体中文](README_CN.md)

A Go reverse-proxy gateway for Cognition's Devin AI, with a React management dashboard. Clients call an OpenAI-compatible `/v1` API; the gateway translates those requests into Connect-RPC calls against `server.codeium.com` and pools multiple Devin accounts behind one endpoint.

The gateway is built on the CLIProxyAPI codebase (Go module `github.com/router-for-me/CLIProxyAPI/v7`, declared in `devin-direct-gateway-src/CLIProxyAPI/go.mod`). This fork's focus is the Devin provider; the codebase still carries the upstream providers for Claude, Codex, Gemini, and others (see `internal/auth/`).

## What it does

- **OpenAI-compatible API surface**: `/v1/chat/completions`, `/v1/completions`, `/v1/responses`, `/v1/messages` (Claude format), and more (`internal/api/server_routes.go:66-79`).
- **Devin protocol translation**: the executor builds `GetChatMessageRequest` protobuf bodies by hand and posts them to `https://server.codeium.com/exa.api_server_pb.ApiServerService/GetChatMessage`, matching the devin-cli client (`CONTEXT.md`, `internal/runtime/executor/helps/devin_wire.go:36-38`).
- **Multi-account pooling**: round-robin (default), weighted round-robin, and fill-first scheduling, with a per-credential cooldown on transient upstream errors (`config.example.yaml:235-261`).
- **Session affinity**: a conversation stays bound to one credential across turns, so upstream session state is not corrupted; TTL and subagent handling are configurable (`internal/config/config_types.go:365-376`, `sdk/cliproxy/auth/conductor_selection.go`).
- **Request-scoped error handling**: Devin trailer errors such as `permission_denied` map to HTTP 403 and fail only the current request; the credential is not cooled down (`internal/api/devin_contract_golden_test.go:374-390`).
- **Tool declaration sanitization**: sensitive-word matching rewrites system prompts and tool descriptions before they reach upstream (`internal/runtime/executor/helps/devin_wire.go:615,663`); observe-only policy rules log matches without altering requests (`config.example.yaml:927-937`).
- **Devin OAuth login**: PKCE browser flow, exposed as the `-devin-login` flag (`internal/auth/devin/pkce.go`, `cmd/server/main.go:142`) and through the dashboard.
- **Web management dashboard**: a single-file React app served at `/management.html` (`internal/api/server_routes.go:54`); the backend also fetches the latest panel into its static directory on demand (`internal/managementasset/updater.go:189`).

## Repository layout

```text
devin_CLIProxyAPI/
├── devin-direct-gateway-src/
│   ├── CLIProxyAPI/                     # Go gateway (module github.com/router-for-me/CLIProxyAPI/v7)
│   │   ├── cmd/server/                  # Entrypoint; -config, -devin-login flags
│   │   ├── internal/
│   │   │   ├── api/                     # HTTP routes, middleware, Devin contract tests
│   │   │   ├── auth/devin/              # Devin OAuth PKCE flow and user status
│   │   │   ├── config/                  # YAML configuration schema
│   │   │   ├── managementasset/         # management.html download and update
│   │   │   ├── registry/                # Model registry and Devin model metadata
│   │   │   └── runtime/executor/        # Devin Connect-RPC executor and wire codecs
│   │   ├── sdk/cliproxy/auth/           # Conductor: credential pool, scheduling, session affinity
│   │   ├── config.example.yaml          # Full configuration template
│   │   ├── CONTEXT.md                   # Devin upstream protocol notes
│   │   └── LICENSE                      # MIT
│   └── Cli-Proxy-API-Management-Center/ # React 19 + Vite dashboard (Bun)
└── README.md
```

## Build and run

### Backend

Requires Go 1.26 or newer (`devin-direct-gateway-src/CLIProxyAPI/go.mod`).

```bash
cd devin-direct-gateway-src/CLIProxyAPI

# Build binary
go build -o cli-proxy-api ./cmd/server

# Copy example config
cp config.example.yaml config.yaml

# Run the proxy server
./cli-proxy-api -config config.yaml
```

The server listens on port `8317` by default (`config.example.yaml`).

### Management dashboard (optional)

Requires [Bun](https://bun.sh/) 1.3+ (`packageManager` field in `package.json`).

```bash
cd devin-direct-gateway-src/Cli-Proxy-API-Management-Center

bun install --frozen-lockfile
bun run build
```

The build emits a single inlined `dist/index.html` (`vite-plugin-singlefile`). Place it as `management.html` in the backend's static directory, or let the backend fetch the latest panel itself; either way it is served at `/management.html`.

## Configuration

Selected keys from `config.yaml` (full template in `devin-direct-gateway-src/CLIProxyAPI/config.example.yaml`):

```yaml
# Devin Provider Configuration
routing:
  strategy: "round-robin"
  session-affinity: true
  session-affinity-ttl: "1h"
  session-affinity-subagents: true

# Forensics and Logging
request-log: true
transient-error-cooldown-seconds: 30

# Request-Scoped Error Handling
oauth-request-scoped-errors:
  devin:
    - status: 403
      match:
        - "permission_denied"
      action: "stop"
    - status: 502
      match:
        - "invalid_argument"
      action: "stop"

# Tool Policy Observation Rules
oauth-tool-policy-observe:
  devin:
    - id: "desc-unrestricted-fs-read"
      match-regexr:
        - "(?i)file ?system[\\s\\S]{0,200}access\\s+any\\s+file"
```

All keys above exist in the config schema (`internal/config/config.go:74-184`, `internal/config/config_types.go:365-376`); the scoped-error and tool-policy blocks are active examples, while `config.example.yaml` ships them commented out.

## Limitations

- The Devin upstream reports errors inside the Connect-RPC end-stream frame, after HTTP 200 is committed. Once a response stream has begun, a failure surfaces mid-stream instead of as a clean HTTP status (`CONTEXT.md:14`).
- Tool-policy observe rules only log matches; the request itself is never altered or rejected (`config.example.yaml:927-928`).
- Session affinity defaults to false; the sample above turns it on explicitly (`config.example.yaml:248`).

## Tests

```bash
cd devin-direct-gateway-src/CLIProxyAPI
go test ./...

# Devin wire-contract tests
go test -v ./internal/api -run TestDevinContract
```

Frontend checks, from `devin-direct-gateway-src/Cli-Proxy-API-Management-Center`:

```bash
bun run test       # bun test
bun run lint       # eslint
bun run type-check # tsc --noEmit
```

## License

This project is released under the MIT License. See [LICENSE](devin-direct-gateway-src/CLIProxyAPI/LICENSE) for details.
