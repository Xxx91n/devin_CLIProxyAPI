# CLI Proxy API 管理中心

一个单文件 Web UI（React + TypeScript），通过 **Management API** 管理 **CLI Proxy API**：配置、凭据、OAuth、配额与日志。

[English](README.md) | [简体中文](README_CN.md)

**主项目**： https://github.com/router-for-me/CLIProxyAPI  
**推荐**：使用最新的后端发布版本（面板资源的同步机制见 `internal/managementasset/updater.go`）

## 范围说明

本仓库只包含 Web 管理界面本身。它通过 CLI Proxy API 的 **Management API**（`/v0/management`）读取/修改配置、管理凭据、查看日志。它不是代理本体，不参与流量转发。

## 快速开始

### 方式 A：使用 CLI Proxy API 自带的 Web UI（推荐）

1. 启动 CLI Proxy API 服务。
2. 打开：`http://<host>:<api_port>/management.html`
3. 输入**管理密钥**并连接。

后端在 API 端口上提供该页面，并自动从 GitHub releases 拉取面板资源。页面会根据当前地址自动推断 API 地址，也支持手动修改。

### 方式 B：开发调试

```bash
bun install --frozen-lockfile
bun run dev
```

打开 `http://localhost:5173`，然后连接到你的 CLI Proxy API 后端实例。

### 方式 C：构建单文件 HTML

```bash
bun install --frozen-lockfile
bun run build
```

- 构建产物：`dist/index.html`（资源已全部内联）。
- CLI Proxy API 打包时，面板资源以 `management.html` 命名发布。
- 本地预览：`bun run preview`

提示：直接用 `file://` 打开 `dist/index.html` 可能遇到浏览器 CORS 限制；更稳妥的方式是用预览/静态服务器打开。

## 连接说明

### API 地址怎么填

以下格式均可，Web UI 会自动归一化：

- `localhost:8317`
- `http://192.168.1.10:8317`
- `https://example.com:8317`
- `http://example.com:8317/v0/management`（后缀会被内部去除）

### 管理密钥（注意：不是 API Keys）

管理密钥会以如下方式随请求发送：

- `Authorization: Bearer <MANAGEMENT_KEY>`（默认）

这与 Web UI 中管理的代理 `api-keys` 不同：后者用于客户端请求代理接口时的鉴权。

### 远程管理

当浏览器不在 localhost 上时，服务端需要开启远程管理：

```yaml
remote-management:
  allow-remote: true
```

管理密钥会作为 Bearer 凭据随每个请求发送。开启 `allow-remote` 之前，请让面板通过 TLS（HTTPS）对外提供，并用 `https://` 访问；明文 HTTP 下，密钥对浏览器与服务器路径上的任何人可见。

管理密钥是同一配置块下的 `secret-key`（明文或 bcrypt 哈希）。完整鉴权规则与服务端限制请参考 CLI Proxy API 服务端文档。

## 功能一览（按页面对应）

- **仪表盘**：连接状态、服务版本/构建时间、关键数量概览、可用模型概览。
- **配置面板**：可视化编辑常用 `config.yaml` 字段、基础设置与代理 `api-keys`；支持 YAML 源码编辑、高亮/搜索与保存前差异预览。
- **AI 提供商**：
  - Gemini/Codex/Claude/Vertex/xAI/Kimi 配置（Base URL、Headers、代理、模型别名、排除模型、Prefix）。
  - OpenAI 兼容提供商（多 Key、Header、从 `/v1/models` 拉取并导入模型别名、可选浏览器侧 `chat/completions` 测试）。
- **认证文件**：上传/下载/删除 JSON 凭据，筛选/搜索/分页，标记 runtime-only；查看单个凭据可用模型（依赖后端支持）；管理 OAuth 排除模型（支持 `*` 通配符）；配置 OAuth 模型别名映射。
- **OAuth**：对 Kimi、Codex、Anthropic/Claude、Antigravity、xAI/Grok 和 Devin 发起 OAuth/设备码流程并轮询状态；支持提交回调 URL；包含 Vertex JSON 凭据导入。
- **配额管理**：管理 Devin、Claude、Codex、Antigravity、Kimi、xAI/Grok 的配额上限与使用情况。
- **插件**：插件页面、已安装插件与插件商店（后端支持插件时显示）。
- **日志**：增量拉取日志、自动刷新、搜索、清空日志；下载请求错误日志文件。
- **系统信息**：快捷链接、版本检查、请求日志开关、本地登录信息清理，以及分组展示的 `/v1/models`。查询模型需要至少一个代理 API Key。

## 技术栈

- React 19 + TypeScript 6.0
- Vite 8（通过 `vite-plugin-singlefile` 实现单文件构建）
- Zustand（状态管理）与 Axios（HTTP 客户端）
- react-router-dom v7（HashRouter）
- CodeMirror 6（YAML 编辑器）、SCSS Modules（样式）
- Motion（动效）、i18next（国际化）

## 多语言支持

支持四种界面语言：

- 英文 (en)
- 简体中文 (zh-CN)
- 繁体中文 (zh-TW)
- 俄文 (ru)

界面语言根据浏览器设置自动检测，也可在登录页或顶部语言菜单手动切换。

## 构建说明

- 构建目标为 `ES2020`，产物是单个 `dist/index.html`，资源全部内联。
- 系统信息页显示的 UI 版本在构建期注入（优先使用环境变量 `VERSION`，否则使用 git tag / `package.json`）。

## 安全提示

- 管理密钥存入浏览器 `localStorage` 时使用轻量混淆格式（`enc::v1::...`），不是明文；仍应视为敏感信息。混淆可逆，不构成安全边界。
- 建议使用独立的浏览器配置或设备进行管理；开启远程管理前请评估暴露面。

## 常见问题

- **无法连接 / 401**：确认 API 地址与管理密钥；非 localhost 访问需要服务端开启 `allow-remote: true`。
- **多次失败后被锁定**：管理密钥连续输错次数过多时，服务端会临时封禁远程 IP。
- **功能提示暂不支持**：后端版本较旧或接口未启用/不存在（如：认证文件模型列表、排除模型、日志相关接口）。
- **OpenAI 提供商测试失败**：测试在浏览器侧执行，受网络与 CORS 影响；这里失败不一定代表服务端不可用。

## 开发命令

```bash
bun run dev        # 启动开发服务器
bun run build      # tsc + Vite 构建
bun run preview    # 本地预览 dist
bun run test       # Bun 测试套件
bun run lint       # ESLint
bun run verify     # 测试 + lint + 构建
bun run format     # Prettier
bun run type-check # tsc --noEmit
```

## 贡献

欢迎提 Issue 与 PR。建议附上：

- 复现步骤（服务端版本 + UI 版本）
- UI 改动截图
- 验证记录（`bun run verify`，以及按需单独运行的 `bun run type-check`）

## 许可证

MIT
