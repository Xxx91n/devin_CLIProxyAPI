# OpenAI Go SDK Realtime 语音示例

[English](README.md) | [简体中文](README_CN.md)

这个示例把语音音频发送给 CLIProxyAPI，并把模型返回的语音回复保存为 WAV 文件。

它使用官方 [`github.com/openai/openai-go/v3`](https://github.com/openai/openai-go) SDK 创建短时效的 Realtime client secret。本示例固定的 SDK 版本（`go.mod` 中的 `v3.50.0`）只提供 Realtime REST 资源，没有 WebSocket 连接辅助方法（`client.Realtime.Connect` 在后续 v3 版本才加入），因此标准的 Realtime 音频事件使用 `github.com/gorilla/websocket` 处理。

## 前置条件

1. 启动 CLIProxyAPI，至少配置一个可用的 ChatGPT/Codex OAuth 凭据。
2. 在 `config.yaml` 中配置代理 API key。
3. 使用 Go 1.26 或更新版本。
4. 准备一个 PCM WAV 文件，参数必须完全符合：
   - 24,000 Hz 采样率
   - 16 位有符号 PCM
   - 单声道
   - 小端序

其他采样率、位深或声道数的 WAV 文件暂不支持。

用 FFmpeg 转换现有录音：

```bash
ffmpeg -i recording.m4a -ar 24000 -ac 1 -c:a pcm_s16le question.wav
```

## 运行

```bash
cd examples/realtime-openai-go

OPENAI_BASE_URL="http://127.0.0.1:8317/v1" \
OPENAI_API_KEY="your-proxy-api-key" \
OPENAI_REALTIME_MODEL="gpt-realtime-2.1" \
OPENAI_REALTIME_INPUT_WAV="question.wav" \
OPENAI_REALTIME_OUTPUT_WAV="response.wav" \
go run .
```

预期输出：

```text
Loaded question.wav (2.4s, 115200 PCM bytes)
Connected to ws://127.0.0.1:8317/v1/realtime?model=gpt-realtime-2.1 using model gpt-realtime-2.1 and voice marin
Sent 2.4s of speech audio
Assistant transcript: The connection is working correctly.
Saved spoken response to response.wav (1.8s, 86400 PCM bytes)
```

播放响应：

```bash
# macOS
afplay response.wav

# Linux
aplay response.wav

# 跨平台可用 FFmpeg
ffplay -autoexit response.wav
```

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `OPENAI_API_KEY` | 是 | — | 为 CLIProxyAPI 配置的 API key。 |
| `OPENAI_REALTIME_INPUT_WAV` | 是 | — | 输入语音 WAV 文件。必须是 24kHz、16 位、单声道 PCM。 |
| `OPENAI_REALTIME_OUTPUT_WAV` | 否 | `response.wav` | 语音回复的输出文件。 |
| `OPENAI_BASE_URL` | 否 | `http://127.0.0.1:8317/v1` | CLIProxyAPI 的 OpenAI 兼容 base URL。URL 没有路径时会补上 `/v1`。 |
| `OPENAI_REALTIME_MODEL` | 否 | `gpt-realtime-2.1` | 标准 Realtime 模型名。CLIProxyAPI 用它连接上游标准 WebSocket，同时在内部选择兼容的 Codex OAuth 凭据。 |
| `OPENAI_REALTIME_VOICE` | 否 | `marin` | Realtime 输出音色。可用音色见 OpenAI 的 [Realtime 文档](https://platform.openai.com/docs/guides/realtime)。 |
| `OPENAI_REALTIME_INSTRUCTIONS` | 否 | `Listen to the user's speech and reply with a short spoken response.` | 附着在 client secret 上的会话指令。 |
| `OPENAI_REALTIME_DEBUG` | 否 | `false` | 打印收到的每一条 Realtime 服务端事件。 |

## 音频流程

1. 官方 OpenAI Go SDK 调用 `POST /v1/realtime/client_secrets`，请求中的音频会话配置为 24kHz PCM 输入和输出。
2. 返回的本地 `ek_...` 凭据用于 `/v1/realtime` WebSocket 认证。
3. 输入 WAV 样本按 200ms 分块，以 `input_audio_buffer.append` 发送。
4. 客户端发送 `input_audio_buffer.commit` 和 `response.create`。
5. Base64 编码的 `response.output_audio.delta` 事件被解码后写入输出 WAV。

CLIProxyAPI 返回的 client secret 只在当前代理实例内有效，不能用于 `api.openai.com`。

## 测试

```bash
go test -race ./...
```

测试会启动进程内的 HTTP/WebSocket 服务器，验证 client secret 配置、输入音频流式发送、输出音频解码和 WAV 生成。
