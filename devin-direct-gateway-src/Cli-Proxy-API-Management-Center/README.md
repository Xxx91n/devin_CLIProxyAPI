# CLI Proxy API Management Center

A single-file Web UI (React + TypeScript) for managing **CLI Proxy API** through its **Management API**: config, credentials, OAuth, quota, and logs.

[English](README.md) | [简体中文](README_CN.md)

**Main Project**: https://github.com/router-for-me/CLIProxyAPI  
**Minimum Required Version**: ≥ 7.2.147 (recommended latest)

## Scope

This repository contains the Web UI only. It talks to the CLI Proxy API **Management API** (`/v0/management`) to read and update config, manage credentials, and view logs. It is not a proxy and does not forward traffic.

## Quick start

### Option A: Use the Web UI bundled in CLI Proxy API (recommended)

1. Start your CLI Proxy API service.
2. Open: `http://<host>:<api_port>/management.html`
3. Enter your **management key** and connect.

The backend serves this page on the API port and downloads the panel asset from GitHub releases automatically. The API address is auto-detected from the page URL; manual override is supported.

### Option B: Run the dev server

```bash
bun install --frozen-lockfile
bun run dev
```

Open `http://localhost:5173`, then connect to your CLI Proxy API backend instance.

### Option C: Build a single HTML file

```bash
bun install --frozen-lockfile
bun run build
```

- Output: `dist/index.html` (all assets inlined).
- For CLI Proxy API bundling, the panel asset is published as `management.html`.
- To preview locally: `bun run preview`

Tip: opening `dist/index.html` via `file://` may be blocked by browser CORS; serving it (preview or a static server) is more reliable.

## Connecting to the server

### API address

Any of these formats works; the UI normalizes them:

- `localhost:8317`
- `http://192.168.1.10:8317`
- `https://example.com:8317`
- `http://example.com:8317/v0/management` (the suffix is stripped internally)

### Management key (not the same as API keys)

The management key is sent with every request as:

- `Authorization: Bearer <MANAGEMENT_KEY>` (default)

This is different from the proxy `api-keys` you manage inside the UI; those authenticate client requests to the proxy endpoints.

### Remote management

When the browser is not on localhost, the server must allow remote management:

```yaml
remote-management:
  allow-remote: true
```

The management key is `secret-key` under the same block (plaintext or bcrypt hashed). Check the CLI Proxy API server documentation for the full authentication rules and server-side limits.

## What you can manage (mapped to the UI pages)

- **Dashboard**: connection status, server version/build date, quick counts, model availability snapshot.
- **Config Panel**: visual editor for common `config.yaml` fields, basic settings, and proxy `api-keys`; YAML source editing with highlighting/search and a save diff preview.
- **AI Providers**:
  - Gemini/Codex/Claude/Vertex/xAI/Kimi key entries (base URL, headers, proxy, model aliases, excluded models, prefix).
  - OpenAI-compatible providers (multiple API keys, custom headers, model alias import via `/v1/models`, optional browser-side `chat/completions` test).
- **Auth Files**: upload/download/delete JSON credentials, filter/search/pagination, runtime-only indicators; view supported models per credential (when the server supports it); manage OAuth excluded models (supports `*` wildcards); configure OAuth model alias mappings.
- **OAuth**: start OAuth/device flows for Kimi, Codex, Anthropic/Claude, Antigravity, xAI/Grok, and Devin; poll status; submit callback URLs; import Vertex JSON credentials.
- **Quota Management**: manage quota limits and usage for Devin, Claude, Codex, Antigravity, Kimi, and xAI/Grok.
- **Plugins**: plugin pages, installed plugins, and plugin store (shown when the backend supports plugins).
- **Logs**: tail logs with incremental polling, auto-refresh, search, and clear; download request error log files.
- **System**: quick links, update check, request logging toggle, local login data cleanup, and a grouped `/v1/models` view. Requires at least one proxy API key to query models.

## Tech stack

- React 19 + TypeScript 6.0
- Vite 8 (single-file build via `vite-plugin-singlefile`)
- Zustand (state management) and Axios (HTTP client)
- react-router-dom v7 (HashRouter)
- CodeMirror 6 (YAML editor), SCSS Modules (styling)
- Motion (animations), i18next (internationalization)

## Languages

Four UI languages are supported:

- English (en)
- Simplified Chinese (zh-CN)
- Traditional Chinese (zh-TW)
- Russian (ru)

The UI language is detected from browser settings and can be switched from the login page or the header language menu.

## Build notes

- Build target is `ES2020`; the output is a single `dist/index.html` with all assets inlined.
- The UI version shown on the System page is injected at build time (env `VERSION`, git tag, or `package.json` fallback).

## Security notes

- The management key is stored in browser `localStorage` in a lightly obfuscated form (`enc::v1::...`) rather than plaintext; treat it as sensitive. The obfuscation is reversible and is not a security boundary.
- Use a dedicated browser profile or device for management, and weigh the exposure before enabling remote management.

## Troubleshooting

- **Can't connect / 401**: confirm the API address and management key; non-localhost access requires `allow-remote: true` on the server.
- **Locked out after repeated failures**: the server temporarily bans remote IPs after too many failed management-key attempts.
- **Some features show "unsupported"**: the backend may be too old or the endpoint is disabled/absent (common for model lists per auth file, excluded models, and logs).
- **OpenAI provider test fails**: the test runs in the browser and depends on network/CORS of the provider endpoint; a failure here does not always mean the server cannot reach it.

## Development

```bash
bun run dev        # Vite dev server
bun run build      # tsc + Vite build
bun run preview    # serve dist locally
bun run test       # Bun test suite
bun run lint       # ESLint
bun run verify     # test + lint + build
bun run format     # Prettier
bun run type-check # tsc --noEmit
```

## Contributing

Issues and PRs are welcome. Please include:

- Reproduction steps (server version + UI version)
- Screenshots for UI changes
- Verification notes (`bun run verify`, plus `bun run type-check` when run separately)

## License

MIT
