# devin_CLIProxyAPI

A high-performance reverse proxy and multi-account credential management gateway for Cognition's Devin AI platform, providing an OpenAI-compatible `/v1` API surface.

## Overview

`devin_CLIProxyAPI` bridges client applications and agentic workflows to the Devin backend (`server.codeium.com`) using Connect-RPC protocol translation. It provides enterprise-ready multi-account pooling, session affinity, intelligent error handling, tool description sanitization, and a modern Web management dashboard.

## Key Features

- **Devin Upstream Integration**: Connect-RPC translation targeting `ApiServerService/GetChatMessage`, impersonating devin-cli client requirements with normalized RFC 4122 Session / Cascade UUIDs.
- **Multi-Account Credential Pooling**: Load-balance multiple Devin accounts with round-robin scheduling and session-specific request pools.
- **Session Affinity**: Sticky credential binding by session ID / message hash across multi-turn interactions, preventing upstream conversation state corruption.
- **Request-Scoped Error Classification**: Automatically isolates request-attributed trailer errors (`invalid_argument`, `permission_denied`) to fail only the immediate request without triggering credential cooldowns or disrupting healthy accounts.
- **Tool Description Sanitization & Policy Observation**: Configurable sentence-level sanitization against sensitive keywords and regex-based policy observation rules to avoid upstream tool declaration rejections.
- **Devin OAuth 2.0 PKCE Flow**: One-click browser login via the management control panel to acquire and hot-register Devin session credentials.
- **Web Management Dashboard**: Embedded Single-Page Application (SPA) for real-time quota tracking, account status inspection, and dynamic configuration.

## Repository Structure

```text
devin_CLIProxyAPI/
├── devin-direct-gateway-src/
│   ├── CLIProxyAPI/                     # Go-based reverse proxy & API gateway
│   │   ├── cmd/server/                  # Main entrypoint
│   │   ├── internal/                    # Core proxy logic, executors, translators, registry
│   │   │   ├── auth/devin/              # Devin OAuth & user status service
│   │   │   ├── runtime/executor/        # Devin Connect-RPC executor & wire serialization
│   │   │   └── registry/                # Conductor, credential pool, session affinity
│   │   ├── config.example.yaml          # Complete configuration template
│   │   └── CONTEXT.md                   # Devin provider protocol specification
│   └── Cli-Proxy-API-Management-Center/ # Frontend management control panel (React 19 + Vite)
│       ├── src/                         # UI features (dashboard, quota, authFiles, providers)
│       └── package.json                 # Bun/Vite build definition
└── README.md
```

## Quick Start

### 1. Build & Run the Backend

Prerequisites: Go 1.23+

```bash
cd devin-direct-gateway-src/CLIProxyAPI

# Build binary
go build -o cli-proxy-api ./cmd/server

# Copy example config
cp config.example.yaml config.yaml

# Run the proxy server
./cli-proxy-api -config config.yaml
```

The proxy starts listening on `http://127.0.0.1:13015` by default.

### 2. Build the Management Center (Optional)

Prerequisites: [Bun](https://bun.sh/) 1.3+

```bash
cd devin-direct-gateway-src/Cli-Proxy-API-Management-Center

bun install --frozen-lockfile
bun run build
```

The production output generates an inlined single-file `dist/index.html`. Rename or place it at `CLIProxyAPI/static/management.html` to serve it directly from the backend via `/management.html`.

## Configuration Highlights

In `config.yaml`:

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

## License

This project is released under the MIT License. See [LICENSE](file:///d:/Aworker/devin_local/devin-direct-gateway-src/CLIProxyAPI/LICENSE) for details.
