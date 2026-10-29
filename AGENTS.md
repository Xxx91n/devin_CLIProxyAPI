# AGENTS.md — Repository Guidance for devin_CLIProxyAPI

## Overview
`devin_CLIProxyAPI` is a monorepo containing:
1. `devin-direct-gateway-src/CLIProxyAPI`: Go 1.23+ reverse-proxy server for Devin AI with multi-account pooling and Connect-RPC translation.
2. `devin-direct-gateway-src/Cli-Proxy-API-Management-Center`: React 19 + TypeScript + Vite management frontend for Devin OAuth and quota administration.

## Component Commands

### Backend (`devin-direct-gateway-src/CLIProxyAPI`)
```bash
cd devin-direct-gateway-src/CLIProxyAPI
gofmt -w .                                          # Format Go files
go build -o cli-proxy-api ./cmd/server               # Build binary
go test ./...                                       # Run all unit tests
go test -v ./internal/api -run TestDevinContract    # Run Devin v1 contract tests
go test -v ./internal/runtime/executor/helps -run TestSanitizeDevin # Run tool description sanitizer tests
```

### Frontend (`devin-direct-gateway-src/Cli-Proxy-API-Management-Center`)
```bash
cd devin-direct-gateway-src/Cli-Proxy-API-Management-Center
bun install --frozen-lockfile                       # Install dependencies
bun run dev                                         # Local dev server
bun run build                                       # Production build (outputs dist/index.html)
bun run test                                        # Run tests
bun run lint                                        # Lint TypeScript/TSX
bun run type-check                                  # Check types
```

## Devin Architecture & Conventions
- Upstream protocol documentation: consult `devin-direct-gateway-src/CLIProxyAPI/CONTEXT.md`.
- Devin trailer errors marked `invalid_argument`, `permission_denied` are request-scoped (`RequestScopedError`). Never cool down credentials on request-scoped errors.
- Session affinity: preserves server-side session continuity by binding downstream sessions to stable credentials.
- Security: Never commit real auth tokens, session tokens, or secrets. Auth credentials belong in `auths/` (gitignored).
