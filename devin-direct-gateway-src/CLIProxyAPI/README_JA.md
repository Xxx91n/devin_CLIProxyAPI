# devin-direct-gateway

[English](README.md) | [简体中文](README_CN.md) | 日本語

[CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) のフォークです。Cognition の Devin を標準的な OpenAI・Anthropic・Gemini API エンドポイント経由で利用できるようにします。上流のプロバイダーはすべて残っています。このフォークが追加・保守しているのは Devin サポートです。

## 概要

- 1 つのローカルサーバーが OpenAI（`/v1/chat/completions`、`/v1/responses`）、Anthropic（`/v1/messages`）、Gemini（`/v1beta`）の各プロトコルを提供し、既存のクライアントや SDK はコード変更なしで接続できます。
- プロバイダーごとに複数アカウントに対応：OAuth ログインフラグまたは設定ファイルの API キーで登録し、リクエストをアカウントプールに分散します（デフォルトはラウンドロビン。`routing.strategy` は weighted-round-robin と fill-first も対応）。
- 提供フォーマット間のプロトコル変換。ストリーミング（SSE）レスポンスと `/v1/ws` の WebSocket ルートを含みます。
- Devin 固有の機能：OAuth PKCE ログイン、Cognition バックエンドへの Connect-RPC 変換、セッションアフィニティ（同じマルチターン会話を同じ資格情報に固定）。プロトコルの詳細は [CONTEXT.md](CONTEXT.md) を参照してください。

## プロバイダー

| プロバイダー | ログインフラグ | 資格情報 |
|---|---|---|
| Devin (Cognition) | `-devin-login` | OAuth。`auth-dir` に保存 |
| Codex (OpenAI) | `-codex-login`、`-codex-device-login` | OAuth または `codex-api-key` |
| Claude | `-claude-login` | OAuth または `claude-api-key` |
| Antigravity (Google Gemini) | `-antigravity-login` | OAuth |
| Kimi | `-kimi-login` | OAuth |
| xAI (Grok) | `-xai-login` | OAuth または `xai-api-key` |
| Meta (Muse Code) | `-meta-login` | OAuth または `meta-api-key` |
| Gemini / Vertex | `-vertex-import` | `gemini-api-key` / `vertex-api-key` |
| OpenAI 互換の任意の上流 | — | `openai-compatibility` エントリ（例：OpenRouter） |

## 要件

- Go 1.26 以上（`go.mod` 参照）、または Docker
- [config.example.yaml](config.example.yaml) をもとに作成した `config.yaml`

## ビルドと実行

```bash
go build -o cli-proxy-api ./cmd/server
./cli-proxy-api -config config.yaml
```

テストの実行：`go test ./...`。

プロバイダーにログインします（プールに入れたいアカウントごとに実行します）：

```bash
./cli-proxy-api -config config.yaml -devin-login
```

サーバーはデフォルトでポート 8317 で待ち受けます（設定の `port`）。

## 設定

`config.yaml` の主な設定項目（全一覧は config.example.yaml を参照）：

| 項目 | 目的 | デフォルト |
|---|---|---|
| `port` | 待ち受けポート | `8317` |
| `api-keys` | クライアントが送る必要のあるキー | — |
| `auth-dir` | OAuth 資格情報の保存先 | `~/.cli-proxy-api` |
| `remote-management.secret-key` | 設定すると Management API を有効化。空なら無効 | 空 |
| `routing.session-affinity` | アフィニティ TTL の間、セッションを 1 つの資格情報に固定 | `false` |
| `proxy-url` | 上流プロキシ（socks5/http/https） | 空 |

## 既知の制限 (Known limitations)

- `api-keys` が config.example.yaml のサンプル値のままサーバーを起動すると、実際のキーに差し替えるまでプロキシ API エンドポイントは無効になります。
- Devin の OAuth リダイレクトは `http://127.0.0.1:<port>/callback` を向く必要があります。リモートサーバーではブラウザーがそのループバックページに届かないため、最終的なコールバック URL を管理パネルに貼り付けてログインを完了させます（docs/management-devin-oauth.md）。

## API エンドポイント

| パス | 形式 |
|---|---|
| `GET /healthz` | ヘルスチェック |
| `GET /v1/models` | OpenAI または Anthropic のモデル一覧（リクエストヘッダーで判別） |
| `POST /v1/chat/completions` | OpenAI Chat Completions |
| `POST /v1/completions` | OpenAI Completions |
| `POST /v1/responses` | OpenAI Responses |
| `POST /v1/messages` | Anthropic Messages |
| `POST /v1/messages/count_tokens` | Anthropic トークン数計算 |
| `POST /v1beta/models/*action` | Gemini generateContent など |
| `POST /v1beta/interactions` | Gemini Interactions |
| `GET /v1/ws` | WebSocket |

## Devin の注意点

- `./cli-proxy-api -config config.yaml -devin-login` で OAuth PKCE フローを実行し、`devin-*.json` 資格情報を `auth-dir` に保存します。
- OAuth コールバックは `/callback`（`/devin/callback` でも可）に到着します。
- 管理パネル経由のブラウザーログインについては [docs/management-devin-oauth.md](docs/management-devin-oauth.md) を参照してください。
- 上流プロトコル（`server.codeium.com` への Connect-RPC、trailer エラー、セッション ID の扱い）は [CONTEXT.md](CONTEXT.md) に文書化されています。
- 任意設定の `devin.sensitive-words` で、システム指示やプロンプト内の指定語を難読化できます。

## Docker

`docker-compose.yml` は公開イメージを実行し、ポート 8317 をマッピングし、`config.yaml`、`auths/`、`logs/`、`plugins/` をマウントします：

```bash
docker compose up -d
```

ソースからイメージをビルドする場合は `./docker-build.sh` を実行してオプション 2 を選んでください。

## Management API と管理パネル

`remote-management.secret-key` を設定すると `/v0/management/*` ルートが有効になります。キー設定時は同梱の Web パネルが `/management.html` で提供され、未設定なら両方とも無効です。パネルのアセットは初回アクセス時に Cli-Proxy-API-Management-Center の GitHub リポジトリからダウンロードされます。

## SDK

`sdk/` 配下の Go SDK で、プロキシを他のプログラムに組み込めます：

- 使い方：[docs/sdk-usage.md](docs/sdk-usage.md)
- エグゼキューターとトランスレーター：[docs/sdk-advanced.md](docs/sdk-advanced.md)
- アクセス制御：[docs/sdk-access.md](docs/sdk-access.md)
- 資格情報の読み込み：[docs/sdk-watcher.md](docs/sdk-watcher.md)
- カスタムプロバイダーの例：`examples/custom-provider`

## 上流とクレジット

このリポジトリは Router-For.ME の [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) のフォークです。管理パネルは [Cli-Proxy-API-Management-Center](https://github.com/router-for-me/Cli-Proxy-API-Management-Center) から来ています。上流 README のスポンサーおよびエコシステムプロジェクトの一覧はここでは転載していません。上流のドキュメント：[help.router-for.me](https://help.router-for.me/)。

## ライセンス

MIT ライセンス。詳細は [LICENSE](LICENSE) を参照してください。
