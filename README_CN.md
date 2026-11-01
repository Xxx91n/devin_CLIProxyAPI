# devin_CLIProxyAPI

[English](README.md) | [简体中文](README_CN.md)

一个面向 Cognition Devin AI 的 Go 反向代理网关，附带 React 管理面板。客户端调用 OpenAI 兼容的 `/v1` API；网关把请求翻译成对 `server.codeium.com` 的 Connect-RPC 调用，并把多个 Devin 账号池化在同一个端点之后。

网关基于 CLIProxyAPI 代码库构建（Go module 为 `github.com/router-for-me/CLIProxyAPI/v7`，见 `devin-direct-gateway-src/CLIProxyAPI/go.mod`）。这个 fork 的重点是 Devin provider；代码库仍保留上游的 Claude、Codex、Gemini 等 provider（见 `internal/auth/`）。

## 功能

- **OpenAI 兼容 API**：`/v1/chat/completions`、`/v1/completions`、`/v1/responses`、`/v1/messages`（Claude 格式）等（`internal/api/server_routes.go:66-79`）。
- **Devin 协议翻译**：executor 手工构造 `GetChatMessageRequest` protobuf 报文，发送到 `https://server.codeium.com/exa.api_server_pb.ApiServerService/GetChatMessage`，对齐 devin-cli 客户端行为（`CONTEXT.md`、`internal/runtime/executor/helps/devin_wire.go:36-38`）。
- **多账号池化**：round-robin（默认）、weighted round-robin、fill-first 三种调度策略，瞬时上游错误触发凭据冷却（`config.example.yaml:235-261`）。
- **会话亲和**：同一会话在多轮交互中固定绑定同一凭据，避免上游会话状态被破坏；TTL 和 subagent 行为可配置（`internal/config/config_types.go:365-376`、`sdk/cliproxy/auth/conductor_selection.go`）。
- **请求级错误处理**：Devin trailer 错误（如 `permission_denied`）映射为 HTTP 403，只让当前请求失败，凭据不会进入冷却（`internal/api/devin_contract_golden_test.go:374-390`）。
- **工具声明清洗**：敏感词匹配会在请求到达上游前改写 system prompt 和工具描述（`internal/runtime/executor/helps/devin_wire.go:615,663`）；observe-only 策略规则只记录命中日志，不改写请求（`config.example.yaml:927-937`）。
- **Devin OAuth 登录**：PKCE 浏览器授权流程，通过 `-devin-login` 命令行标志（`internal/auth/devin/pkce.go`、`cmd/server/main.go:142`）或管理面板触发。
- **Web 管理面板**：单文件 React 应用，由 `/management.html` 提供（`internal/api/server_routes.go:54`）；后端也会按需把最新面板拉取到 static 目录（`internal/managementasset/updater.go:189`）。

## 仓库结构

```text
devin_CLIProxyAPI/
├── devin-direct-gateway-src/
│   ├── CLIProxyAPI/                     # Go 网关（module github.com/router-for-me/CLIProxyAPI/v7）
│   │   ├── cmd/server/                  # 入口；-config、-devin-login 标志
│   │   ├── internal/
│   │   │   ├── api/                     # HTTP 路由、中间件、Devin 契约测试
│   │   │   ├── auth/devin/              # Devin OAuth PKCE 流程与用户状态
│   │   │   ├── config/                  # YAML 配置结构
│   │   │   ├── managementasset/         # management.html 下载与更新
│   │   │   ├── registry/                # 模型注册表与 Devin 模型元数据
│   │   │   └── runtime/executor/        # Devin Connect-RPC executor 与线路编解码
│   │   ├── sdk/cliproxy/auth/           # Conductor：凭据池、调度、会话亲和
│   │   ├── config.example.yaml          # 完整配置模板
│   │   ├── CONTEXT.md                   # Devin 上游协议说明
│   │   └── LICENSE                      # MIT
│   └── Cli-Proxy-API-Management-Center/ # React 19 + Vite 管理面板（Bun）
└── README.md
```

## 构建与运行

### 后端

需要 Go 1.26 或更新版本（`devin-direct-gateway-src/CLIProxyAPI/go.mod`）。

```bash
cd devin-direct-gateway-src/CLIProxyAPI

# Build binary
go build -o cli-proxy-api ./cmd/server

# Copy example config
cp config.example.yaml config.yaml
```

启动服务端之前，先补全复制出来的 `config.yaml`：

1. 把 `api-keys` 的占位值（`your-api-key-1` … `your-api-key-3`，`config.example.yaml:40-42`）换成真实 key。占位值仍在时，示例 key 安全模式会禁用代理 API 端点并退出服务端（`cmd/server/main.go:653-662`）。
2. 使用管理面板时设置 `remote-management.secret-key`。不设置（也没有环境变量兜底）时管理密钥缺失，Management API 不会启用（`internal/api/server.go:235`）。
3. 执行一次 `./cli-proxy-api -config config.yaml -devin-login`，把 Devin OAuth 凭据保存到 `auth-dir`；没有该凭据 Devin 请求无法工作。

```bash
# Run the proxy server
./cli-proxy-api -config config.yaml
```

服务默认监听 `8317` 端口（`config.example.yaml`）。

### 管理面板（可选）

需要 [Bun](https://bun.sh/) 1.3+（`package.json` 的 `packageManager` 字段）。

```bash
cd devin-direct-gateway-src/Cli-Proxy-API-Management-Center

bun install --frozen-lockfile
bun run build
```

构建产物是单个内联的 `dist/index.html`（`vite-plugin-singlefile`）。把它作为 `management.html` 放进后端的 static 目录即可，或者让后端自行拉取最新面板；两种方式都通过 `/management.html` 访问。

## 配置

`config.yaml` 中的部分键（完整模板见 `devin-direct-gateway-src/CLIProxyAPI/config.example.yaml`）：

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

以上键均存在于配置结构中（`internal/config/config.go:74-184`、`internal/config/config_types.go:365-376`）；scoped-error 和 tool-policy 块是有效示例，`config.example.yaml` 中以注释形式给出。

## 局限性

- Devin 上游在 Connect-RPC end-stream 帧内、即 HTTP 200 已提交之后才报告错误。响应流一旦开始，失败会以流中错误的形式出现，而不是干净的 HTTP 状态码（`CONTEXT.md:14`）。
- Tool-policy observe 规则只记录命中日志，请求本身不会被改写或拒绝（`config.example.yaml:927-928`）。
- 会话亲和默认关闭；上面的示例把它显式打开（`config.example.yaml:248`）。

## 测试

```bash
cd devin-direct-gateway-src/CLIProxyAPI
go test ./...

# Devin wire-contract tests
go test -v ./internal/api -run TestDevinContract
```

前端检查，在 `devin-direct-gateway-src/Cli-Proxy-API-Management-Center` 下执行：

```bash
bun run test       # bun test
bun run lint       # eslint
bun run type-check # tsc --noEmit
```

## 许可证

本项目以 MIT 许可证发布，详见 [LICENSE](devin-direct-gateway-src/CLIProxyAPI/LICENSE)。
