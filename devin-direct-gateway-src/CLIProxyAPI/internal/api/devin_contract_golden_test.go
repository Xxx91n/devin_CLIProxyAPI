package api

// Contract golden base for the Devin provider /v1 surface.
// Each test drives the real pipeline — gin router -> conductor -> devin executor
// -> mock Connect-RPC upstream — and pins the exact downstream wire shapes
// expected by consumers: the non-stream chat.completion structure, the SSE frame
// ordering (including the terminal [DONE]), and the error-envelope triad
// (request-scoped 4xx passthrough, 503 auth_unavailable, channel 5xx).
// Any drift in these shapes must turn this file red.

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"google.golang.org/protobuf/encoding/protowire"
)

// --- Connect-RPC wire helpers (recorded upstream shapes) ---

// devinContractFrame wraps a payload in a Connect-RPC envelope:
// 1 flag byte + 4-byte big-endian length + payload.
func devinContractFrame(flag byte, payload []byte) []byte {
	frame := make([]byte, 5+len(payload))
	frame[0] = flag
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)
	return frame
}

func devinContractTagBytes(b []byte, num protowire.Number, s string) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendString(b, s)
}

func devinContractTagVarint(b []byte, num protowire.Number, v uint64) []byte {
	b = protowire.AppendTag(b, num, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

// devinContractUsageField encodes the field-7 usage submessage
// (2=prompt, 3=completion, 5=cache-read tokens).
func devinContractUsageField(prompt, completion, cached uint64) []byte {
	var u []byte
	u = devinContractTagVarint(u, 2, prompt)
	u = devinContractTagVarint(u, 3, completion)
	u = devinContractTagVarint(u, 5, cached)
	return u
}

// devinContractDataFrame builds one GetChatMessageResponse data frame.
// Field map per helps.ParseDevinFrame: 1=outputID, 3=content delta,
// 5=stop reason, 7=usage submessage, 9=thinking delta, 17=messageID.
func devinContractDataFrame(content, thinking string, usage []byte, stopReason uint64) []byte {
	var p []byte
	p = devinContractTagBytes(p, 1, "out-contract-1")
	p = devinContractTagBytes(p, 17, "msg-contract-1")
	if thinking != "" {
		p = devinContractTagBytes(p, 9, thinking)
	}
	if content != "" {
		p = devinContractTagBytes(p, 3, content)
	}
	if usage != nil {
		p = protowire.AppendTag(p, 7, protowire.BytesType)
		p = protowire.AppendBytes(p, usage)
	}
	if stopReason != 0 {
		p = devinContractTagVarint(p, 5, stopReason)
	}
	return devinContractFrame(0x00, p)
}

func devinContractEOS(errJSON string) []byte {
	if errJSON == "" {
		errJSON = "{}"
	}
	return devinContractFrame(0x02, []byte(errJSON))
}

// newDevinContractUpstream serves canned Connect-RPC GetChatMessage responses.
// The user prompt text travels verbatim inside the request protobuf, so TRIG_*
// markers select the injected trailer error — the same mechanism the WSL arm64
// closed loop used.
func newDevinContractUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var stream bytes.Buffer
		switch {
		case bytes.Contains(body, []byte("TRIG_PD")):
			stream.Write(devinContractEOS(`{"error":{"code":"permission_denied","message":"mock org policy denied (contract)"}}`))
		case bytes.Contains(body, []byte("TRIG_UA")):
			stream.Write(devinContractEOS(`{"error":{"code":"unauthenticated","message":"mock session token expired (contract)"}}`))
		case bytes.Contains(body, []byte("TRIG_INTERNAL")):
			stream.Write(devinContractEOS(`{"error":{"code":"internal","message":"mock internal fault (contract)"}}`))
		default:
			stream.Write(devinContractDataFrame("Hello", "think-part-1 ", nil, 0))
			stream.Write(devinContractDataFrame("world", "think-part-2", devinContractUsageField(11, 6, 468), 1))
			stream.Write(devinContractEOS(""))
		}
		w.Header().Set("Content-Type", "application/connect+proto")
		w.Header().Set("Connect-Protocol-Version", "1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(stream.Bytes())
	}))
}

// newDevinContractServer wires a real devin executor + credential + registered
// model into the standard api test server.
func newDevinContractServer(t *testing.T, upstreamURL string) *Server {
	t.Helper()
	server := newTestServer(t)
	server.handlers.AuthManager.RegisterExecutor(executor.NewDevinExecutor(server.cfg))
	credential := &auth.Auth{
		ID:       "devin-contract-auth",
		Provider: "devin",
		Status:   auth.StatusActive,
		Attributes: map[string]string{
			"api_key":  "devin-contract-token",
			"base_url": upstreamURL,
		},
	}
	if _, err := server.handlers.AuthManager.Register(context.Background(), credential); err != nil {
		t.Fatalf("register devin auth: %v", err)
	}
	registry.GetGlobalRegistry().RegisterClient(credential.ID, credential.Provider, []*registry.ModelInfo{{ID: "devin/swe-2"}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(credential.ID)
	})
	return server
}

func devinContractChat(t *testing.T, server *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")
	server.engine.ServeHTTP(rr, req)
	return rr
}

// --- assertion helpers ---

func devinContractJSON(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("response is not JSON: %v; body=%s", err, rr.Body.String())
	}
	return m
}

func devinContractKeys(t *testing.T, obj map[string]any, path string, want ...string) {
	t.Helper()
	if len(obj) != len(want) {
		t.Fatalf("%s: key count = %d, want %d; got keys %v", path, len(obj), len(want), devinContractKeyList(obj))
	}
	for _, k := range want {
		if _, ok := obj[k]; !ok {
			t.Fatalf("%s: missing key %q; got keys %v", path, k, devinContractKeyList(obj))
		}
	}
}

func devinContractKeyList(obj map[string]any) []string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys
}

// devinContractSSE splits an SSE body into data payloads; the last element of a
// well-formed stream is the literal "[DONE]".
func devinContractSSE(t *testing.T, body string) []string {
	t.Helper()
	var out []string
	for _, block := range strings.Split(body, "\n\n") {
		block = strings.TrimRight(block, "\n")
		if block == "" {
			continue
		}
		if !strings.HasPrefix(block, "data: ") {
			t.Fatalf("SSE block missing 'data: ' prefix: %q", block)
		}
		out = append(out, strings.TrimPrefix(block, "data: "))
	}
	return out
}

// --- contract tests ---

// Non-stream chat.completion: exact top-level/choice/message/usage key sets and
// value semantics as consumed by clients.
func TestDevinContractNonStreamShape(t *testing.T) {
	upstream := newDevinContractUpstream(t)
	defer upstream.Close()
	server := newDevinContractServer(t, upstream.URL)

	rr := devinContractChat(t, server, `{"model":"devin/swe-2","messages":[{"role":"user","content":"hi"}]}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	m := devinContractJSON(t, rr)
	devinContractKeys(t, m, "root", "id", "object", "created", "model", "choices", "usage")
	if id, _ := m["id"].(string); !strings.HasPrefix(id, "interaction_") {
		t.Fatalf("id = %q, want interaction_* prefix", id)
	}
	if m["object"] != "chat.completion" {
		t.Fatalf("object = %v, want chat.completion", m["object"])
	}
	if _, ok := m["created"].(float64); !ok {
		t.Fatalf("created is not a number: %v", m["created"])
	}
	if m["model"] != "devin/swe-2" {
		t.Fatalf("model = %v, want devin/swe-2", m["model"])
	}

	choices, ok := m["choices"].([]any)
	if !ok || len(choices) != 1 {
		t.Fatalf("choices shape wrong: %v", m["choices"])
	}
	choice := choices[0].(map[string]any)
	devinContractKeys(t, choice, "choices[0]", "index", "message", "finish_reason")
	if choice["index"] != float64(0) {
		t.Fatalf("index = %v, want 0", choice["index"])
	}
	if choice["finish_reason"] != "stop" {
		t.Fatalf("finish_reason = %v, want stop", choice["finish_reason"])
	}

	msg := choice["message"].(map[string]any)
	devinContractKeys(t, msg, "message", "role", "content", "reasoning_content")
	if msg["role"] != "assistant" {
		t.Fatalf("role = %v, want assistant", msg["role"])
	}
	if msg["content"] != "Helloworld" {
		t.Fatalf("content = %v, want Helloworld", msg["content"])
	}
	if msg["reasoning_content"] != "think-part-1 think-part-2" {
		t.Fatalf("reasoning_content = %v", msg["reasoning_content"])
	}

	usage := m["usage"].(map[string]any)
	devinContractKeys(t, usage, "usage", "prompt_tokens", "completion_tokens", "total_tokens", "prompt_tokens_details")
	// Devin merges uncached prompt tokens + cache-read tokens into prompt_tokens.
	if usage["prompt_tokens"] != float64(479) {
		t.Fatalf("prompt_tokens = %v, want 479", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != float64(6) {
		t.Fatalf("completion_tokens = %v, want 6", usage["completion_tokens"])
	}
	if usage["total_tokens"] != float64(485) {
		t.Fatalf("total_tokens = %v, want 485", usage["total_tokens"])
	}
	ptd := usage["prompt_tokens_details"].(map[string]any)
	devinContractKeys(t, ptd, "prompt_tokens_details", "cached_tokens")
	if ptd["cached_tokens"] != float64(468) {
		t.Fatalf("cached_tokens = %v, want 468", ptd["cached_tokens"])
	}
}

// Stream: SSE wire sequence — role chunk first, reasoning/content deltas in
// upstream frame order, terminal finish chunk carrying usage, then [DONE].
func TestDevinContractStreamFrameOrder(t *testing.T) {
	upstream := newDevinContractUpstream(t)
	defer upstream.Close()
	server := newDevinContractServer(t, upstream.URL)

	rr := devinContractChat(t, server, `{"model":"devin/swe-2","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"hi"}]}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	frames := devinContractSSE(t, rr.Body.String())
	if len(frames) != 7 {
		t.Fatalf("SSE data frames = %d, want 7 (role + 4 deltas + finish + DONE); body=%q", len(frames), rr.Body.String())
	}
	if frames[len(frames)-1] != "[DONE]" {
		t.Fatalf("last SSE frame = %q, want [DONE]", frames[len(frames)-1])
	}
	if !strings.HasSuffix(rr.Body.String(), "data: [DONE]\n\n") {
		t.Fatalf("stream does not end with 'data: [DONE]\\n\\n'")
	}

	type chunk struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int             `json:"index"`
			Delta        json.RawMessage `json:"delta"`
			FinishReason *string         `json:"finish_reason"`
		} `json:"choices"`
		Usage map[string]any `json:"usage"`
	}
	parse := func(i int) chunk {
		t.Helper()
		var c chunk
		if err := json.Unmarshal([]byte(frames[i]), &c); err != nil {
			t.Fatalf("frame %d not JSON: %v", i, err)
		}
		if c.Object != "chat.completion.chunk" {
			t.Fatalf("frame %d object = %q", i, c.Object)
		}
		if c.Model != "devin/swe-2" {
			t.Fatalf("frame %d model = %q", i, c.Model)
		}
		if !strings.HasPrefix(c.ID, "interaction_") {
			t.Fatalf("frame %d id = %q", i, c.ID)
		}
		if len(c.Choices) != 1 || c.Choices[0].Index != 0 {
			t.Fatalf("frame %d choices malformed: %s", i, frames[i])
		}
		return c
	}

	// Frame 0: role announcement only.
	c0 := parse(0)
	if string(c0.Choices[0].Delta) != `{"role":"assistant"}` {
		t.Fatalf("frame 0 delta = %s", c0.Choices[0].Delta)
	}
	if c0.Choices[0].FinishReason != nil {
		t.Fatalf("frame 0 finish_reason = %v, want null", *c0.Choices[0].FinishReason)
	}

	// Deltas preserve upstream order: think1, Hello, think2, world.
	wantDeltas := []string{
		`{"reasoning_content":"think-part-1 "}`,
		`{"content":"Hello"}`,
		`{"reasoning_content":"think-part-2"}`,
		`{"content":"world"}`,
	}
	for i, want := range wantDeltas {
		c := parse(i + 1)
		if string(c.Choices[0].Delta) != want {
			t.Fatalf("frame %d delta = %s, want %s", i+1, c.Choices[0].Delta, want)
		}
		if c.Choices[0].FinishReason != nil {
			t.Fatalf("frame %d finish_reason = %v, want null", i+1, *c.Choices[0].FinishReason)
		}
	}

	// Terminal chunk: empty delta, finish_reason=stop, full usage block.
	cf := parse(5)
	if string(cf.Choices[0].Delta) != `{}` {
		t.Fatalf("finish chunk delta = %s, want {}", cf.Choices[0].Delta)
	}
	if cf.Choices[0].FinishReason == nil || *cf.Choices[0].FinishReason != "stop" {
		t.Fatalf("finish chunk finish_reason = %v, want stop", cf.Choices[0].FinishReason)
	}
	devinContractKeys(t, cf.Usage, "stream usage", "prompt_tokens", "completion_tokens", "total_tokens", "prompt_tokens_details")
}

// Error envelope, request-scoped arm: a permission_denied trailer maps to
// HTTP 403 + permission_error envelope, and the credential is NOT cooled —
// the immediately following request still succeeds (ADR-0001 core guarantee).
func TestDevinContractErrorRequestScoped(t *testing.T) {
	upstream := newDevinContractUpstream(t)
	defer upstream.Close()
	server := newDevinContractServer(t, upstream.URL)

	rr := devinContractChat(t, server, `{"model":"devin/swe-2","messages":[{"role":"user","content":"TRIG_PD"}]}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rr.Code, rr.Body.String())
	}
	m := devinContractJSON(t, rr)
	devinContractKeys(t, m, "error root", "error")
	e := m["error"].(map[string]any)
	devinContractKeys(t, e, "error", "message", "type", "code")
	if !strings.HasPrefix(e["message"].(string), "devin upstream error (permission_denied): ") {
		t.Fatalf("error message = %v", e["message"])
	}
	if e["type"] != "permission_error" {
		t.Fatalf("error type = %v, want permission_error", e["type"])
	}
	if e["code"] != "insufficient_quota" {
		t.Fatalf("error code = %v, want insufficient_quota", e["code"])
	}

	// Request-scoped: no cooldown — next baseline request returns 200.
	rr2 := devinContractChat(t, server, `{"model":"devin/swe-2","messages":[{"role":"user","content":"hi"}]}`)
	if rr2.Code != http.StatusOK {
		t.Fatalf("post-request-scoped status = %d, want 200; body=%s", rr2.Code, rr2.Body.String())
	}
}

// Error envelope, channel arm: an internal trailer maps to HTTP 502 +
// server_error envelope. With a single credential the channel transient
// cooldown then surfaces as 503 auth_unavailable on the next request.
func TestDevinContractErrorChannel5xx(t *testing.T) {
	upstream := newDevinContractUpstream(t)
	defer upstream.Close()
	server := newDevinContractServer(t, upstream.URL)

	rr := devinContractChat(t, server, `{"model":"devin/swe-2","messages":[{"role":"user","content":"TRIG_INTERNAL"}]}`)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", rr.Code, rr.Body.String())
	}
	m := devinContractJSON(t, rr)
	e := m["error"].(map[string]any)
	devinContractKeys(t, e, "error", "message", "type", "code")
	if !strings.HasPrefix(e["message"].(string), "devin upstream error (internal): ") {
		t.Fatalf("error message = %v", e["message"])
	}
	if e["type"] != "server_error" {
		t.Fatalf("error type = %v, want server_error", e["type"])
	}
	if e["code"] != "internal_server_error" {
		t.Fatalf("error code = %v, want internal_server_error", e["code"])
	}

	// Single-credential pool: channel cooldown => next request 503s fast.
	rr2 := devinContractChat(t, server, `{"model":"devin/swe-2","messages":[{"role":"user","content":"hi"}]}`)
	if rr2.Code != http.StatusServiceUnavailable {
		t.Fatalf("post-channel-error status = %d, want 503; body=%s", rr2.Code, rr2.Body.String())
	}
	m2 := devinContractJSON(t, rr2)
	e2 := m2["error"].(map[string]any)
	devinContractKeys(t, e2, "error", "message", "type", "code")
	if !strings.Contains(e2["message"].(string), "auth_unavailable") {
		t.Fatalf("503 message missing auth_unavailable: %v", e2["message"])
	}
	if e2["type"] != "server_error" || e2["code"] != "internal_server_error" {
		t.Fatalf("transient 503 type/code = %v/%v", e2["type"], e2["code"])
	}
}

// Error envelope, credential arm: an unauthenticated trailer maps to HTTP 401,
// cools the credential, and the next request surfaces 503 auth_unavailable.
func TestDevinContractErrorAuthUnavailable(t *testing.T) {
	upstream := newDevinContractUpstream(t)
	defer upstream.Close()
	server := newDevinContractServer(t, upstream.URL)

	rr := devinContractChat(t, server, `{"model":"devin/swe-2","messages":[{"role":"user","content":"TRIG_UA"}]}`)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rr.Code, rr.Body.String())
	}
	m := devinContractJSON(t, rr)
	e := m["error"].(map[string]any)
	devinContractKeys(t, e, "error", "message", "type", "code")
	if !strings.HasPrefix(e["message"].(string), "devin upstream error (unauthenticated): ") {
		t.Fatalf("error message = %v", e["message"])
	}
	if e["type"] != "authentication_error" {
		t.Fatalf("error type = %v, want authentication_error", e["type"])
	}
	if e["code"] != "invalid_api_key" {
		t.Fatalf("error code = %v, want invalid_api_key", e["code"])
	}

	rr2 := devinContractChat(t, server, `{"model":"devin/swe-2","messages":[{"role":"user","content":"hi"}]}`)
	if rr2.Code != http.StatusServiceUnavailable {
		t.Fatalf("post-auth-error status = %d, want 503; body=%s", rr2.Code, rr2.Body.String())
	}
	m2 := devinContractJSON(t, rr2)
	e2 := m2["error"].(map[string]any)
	// Terminal credential cooldown surfaces a distinct auth_unavailable flavor:
	// authentication_error + upstream_authentication_required + retryable=false.
	devinContractKeys(t, e2, "error", "message", "type", "code", "retryable")
	if !strings.HasPrefix(e2["message"].(string), "auth_unavailable: no auth available") {
		t.Fatalf("503 message = %v", e2["message"])
	}
	if e2["type"] != "authentication_error" || e2["code"] != "upstream_authentication_required" {
		t.Fatalf("terminal 503 type/code = %v/%v", e2["type"], e2["code"])
	}
	if e2["retryable"] != false {
		t.Fatalf("terminal 503 retryable = %v, want false", e2["retryable"])
	}
}
