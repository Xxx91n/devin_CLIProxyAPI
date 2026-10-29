# CLIProxyAPI (Devin Provider)

CLIProxyAPI extension providing a reverse-proxy and multi-account manager for Cognition's Devin backend.

## Language

### Upstream protocol

**Devin Upstream**:
Cognition's backend at server.codeium.com, reached over Connect-RPC (`ApiServerService/GetChatMessage`) with hand-built protobuf bodies that impersonate the devin-cli client.
_Avoid_: backend (this service is itself a backend), model API

**Trailer Error**:
The authoritative error channel of the Devin Upstream: a JSON error object inside the Connect-RPC end-stream frame, arriving after HTTP 200 was already committed.
_Avoid_: HTTP error, status error

**Session ID / Cascade ID**:
Upstream conversation identifiers carried in request fields 15/16. Always normalized to RFC 4122 UUIDs before going on the wire; non-UUID downstream session strings map deterministically via UUIDv5.
_Avoid_: conversation id, thread id

### Pool and governance

**Credential Pool**:
The set of per-account JSON files under `auth-dir`; each file is one account slot, hot-registered by a file watcher and load-balanced by the conductor (round-robin by default).
_Avoid_: account list, keyring

**Credential Cooldown**:
Temporary exclusion of one credential slot from selection, driven only by explicit credential-class evidence (unauthenticated, quota exhaustion). Scoped per credential, never per provider.
_Avoid_: circuit breaker (a channel concept), ban, disable

**Channel Health**:
The transport/upstream-side failure signal (5xx, timeouts, connection failures, overload). Measured independently of credential state; conflating the two was the 2026-09-16 amplification bug.
_Avoid_: upstream status, availability

**Request-Scoped Error**:
A failure attributable to the request itself (bad arguments, permission denied, policy). Fails that request only — never triggers Credential Cooldown or cross-credential retry.
_Avoid_: client error (broader), fatal error

**Session Affinity**:
Sticky binding of a downstream session identifier to one credential slot for the affinity TTL, so multi-turn conversations keep server-side session state coherent.
_Avoid_: sticky session (implies load balancer), pinning

### Boundaries

**V1 API Contract**:
The downstream /v1/* HTTP surface providing standard OpenAI-compatible endpoints. Response shapes, status codes, SSE framing and error envelopes must remain strictly compatible with client expectations.
_Avoid_: private API, internal endpoint

**Upstream Incident Window**:
A bounded interval in which the Devin Upstream returns inconsistent request-level errors on identical request shapes. Distinguished from request-shape bugs by prior success on the same shape.
_Avoid_: outage (implies total unavailability), bug
