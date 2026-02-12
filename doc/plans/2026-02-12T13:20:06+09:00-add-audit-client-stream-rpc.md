# Add Audit Client-Streaming RPC

## Context

Every hook invocation currently only sends a `RequestPermission` unary RPC to the server. There is no way for the server to observe/log all hook events for audit purposes. We want to add a client-streaming `Audit` RPC so every hook invocation streams its data to the server, where it can be logged or discarded based on configuration.

Since the `crabhook` client is a short-lived process (one per hook invocation), each stream will contain exactly one message. Client-streaming is chosen over unary for future extensibility (a long-lived daemon client could reuse the stream).

## Plan

### 1. Proto changes

**File:** `hook/api/scheme/proto/permission/v1/permission.proto`

Add to `PermissionService`:
```protobuf
rpc Audit(stream AuditEvent) returns (AuditResponse);
```

Add messages:
```protobuf
message AuditEvent {
  PermissionRequest request = 1;
  google.protobuf.Timestamp timestamp = 2;
}

message AuditResponse {
  int32 events_received = 1;
  bool success = 2;
  string message = 3;
}
```

Import `google/protobuf/timestamp.proto`. Reuses `PermissionRequest` since it already carries all hook input data.

Note: Each stream currently contains a single event from one hook invocation. Client-streaming is chosen over unary for future extensibility (long-lived daemon client). This should be documented in proto comments.

### 2. Regenerate protobuf code

```bash
cd hook && buf generate
```

Regenerates `permission.pb.go` and `permission_grpc.pb.go`. Then update `go.mod` if timestamp proto pulls new deps.

### 3. Audit handler interface + implementations

**New file:** `hook/internal/server/audit.go`

```go
type AuditHandler interface {
    HandleAuditEvent(ctx context.Context, event *pb.AuditEvent) error
    Close() error
}
```

Two implementations:
- **`NoOpAuditHandler`** — discards events (default when audit disabled)
- **`LogAuditHandler`** — writes to an `io.Writer` in `"text"` or `"json"` format, mutex-protected
  - Text: `[timestamp] event | tool | session=... msg=...`
  - JSON: `protojson.Marshal(event)` per line
  - `Close()` uses `sync.Once` for idempotency; only closes writer if it's a file (not stderr/stdout)

### 4. Service impl layer

**File:** `hook/api/impl/go/permission/v1/service.go`

- Add `AuditHandler` interface (mirrors `server.AuditHandler` to avoid import cycle)
- Add `auditHandler` field to `Service`
- Update `NewService` to accept audit handler
- Implement `Audit(stream)` method: recv loop → delegate to handler → `SendAndClose` with count

### 5. Server wiring

**File:** `hook/internal/server/server.go`

- Add `AuditHandler` field to `Config` and `Server`
- Default to `NoOpAuditHandler` if nil
- Pass to `impl.NewService(server, auditHandler)`
- Call `auditHandler.Close()` in `Stop()`

### 6. Client changes

**File:** `hook/cmd/crabhook/internal/root.go`

- Add `sendAuditEvent(ctx, client, req)` function that:
  1. Opens audit stream with 5s timeout
  2. Sends one `AuditEvent` with `timestamppb.Now()`
  3. Calls `CloseAndRecv()`
  4. Logs warnings to stderr on failure

- Call `sendAuditEvent` **synchronously after** `RequestPermission` (and response output):
  - Audit is best-effort and should not add latency to the permission critical path
  - Short 5s timeout prevents hanging on exit
  - Failure doesn't abort the hook (warnings to stderr only)

- Add `import "google.golang.org/protobuf/types/known/timestamppb"`

### 7. Serve command flags

**File:** `hook/cmd/crabhook/internal/serve.go`

New flags:
- `--audit-enable` (bool, default false) — enable audit logging
- `--audit-output` (string, default "stderr") — "stderr" or file path
- `--audit-format` (string, default "text") — "text" or "json"

When `--audit-enable`, create `LogAuditHandler` with appropriate writer. Otherwise, leave `Config.AuditHandler` nil (server defaults to NoOp).

### 8. Tests

**`hook/internal/server/audit_test.go`:**
- `TestNoOpAuditHandler` — no error, no output
- `TestLogAuditHandler_Text` — verify text output contains event, tool, session
- `TestLogAuditHandler_JSON` — verify valid JSON output with expected fields

**`hook/internal/server/prompt_logic_test.go`:** (no changes needed)

**`hook/api/impl/go/permission/v1/service_test.go`:**
- `TestServiceAudit` — mock stream with 2 events, verify handler receives both, response has count=2

## Files to modify

| File | Action |
|------|--------|
| `hook/api/scheme/proto/permission/v1/permission.proto` | Add Audit RPC + messages |
| `hook/api/gen/go/permission/v1/permission.pb.go` | Regenerate |
| `hook/api/gen/go/permission/v1/permission_grpc.pb.go` | Regenerate |
| `hook/internal/server/audit.go` | **New** — handler interface + impls |
| `hook/api/impl/go/permission/v1/service.go` | Add Audit impl + audit handler |
| `hook/internal/server/server.go` | Wire audit handler |
| `hook/cmd/crabhook/internal/root.go` | Send audit events |
| `hook/cmd/crabhook/internal/serve.go` | Add audit flags |
| `hook/internal/server/audit_test.go` | **New** — handler tests |
| `hook/api/impl/go/permission/v1/service_test.go` | **New** — service audit tests |

## Verification

1. `cd hook && buf generate` — regenerate proto
2. `go build ./...` — compile
3. `go test ./hook/...` — all tests pass
4. Manual: `crabhook serve --audit-enable` then `echo '{"hook_event_name":"PreToolUse","tool_name":"Bash","session_id":"s1"}' | crabhook` — verify audit line on server stderr
5. Manual: `crabhook serve --audit-enable --audit-format=json --audit-output=/tmp/audit.log` — verify JSON in file
