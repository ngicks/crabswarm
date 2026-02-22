# Split Hook Audit: Interface vs Implementation

## Context

`cmd/crabswarm/commands/hook_audit.go` currently contains both the cobra command definition and all business logic (stdin reading, JSON parsing, gRPC client creation, audit event sending). This makes it untestable and couples process-level concerns (`os.Stdin`) with business logic. The user has scaffolded `pkg/crabswarm/hook.go` and `pkg/crabswarm/server.go` as targets for the extracted implementation. The `io.Pipe` cancellable-stdin pattern currently in `hook_audit.go` should be extracted to a shared utility so future hook subcommands can reuse it.

## Plan

### 1. Create `cmd/internal/stdiopipe/stdiopipe.go` — cancellable reader helper

Internal utility package under `cmd/` for hook subcommands.

```go
package stdiopipe

// Stdin returns an [io.ReadCloser] backed by [os.Stdin] through an [io.Pipe].
//
// This is necessary because Read call on [os.Stdin] cannot be unblocked by closing it.
//
// Only one invocation is allowed to be called per one process.
func Stdin(ctx context.Context) io.ReadCloser
```

- Uses `sync.Once` internally; panics on second call (only one stdin consumer is valid)
- Always reads from `os.Stdin` (no `src` parameter)
- Returns `io.ReadCloser` (`*io.PipeReader` satisfies this) — caller uses `Close()` to clean up
- Spawns goroutine: `<-ctx.Done()` → `pr.CloseWithError(ctx.Err())`
- Spawns goroutine: `io.Copy(pw, os.Stdin)` — on copy error, calls `pw.CloseWithError(err)` to propagate read errors; on clean EOF, calls `pw.Close()`

### 2. Write `pkg/crabswarm/hook.go` — extracted audit hook logic

```go
// HookAudit reads a PreToolUseHookInput from r and sends it as an audit event.
func HookAudit(ctx context.Context, r io.Reader, client pb.AuditServiceClient) error
```

- Standalone function (hook is oneshot — no state to hold across calls)
- Accepts `io.Reader` (not `os.Stdin`) — testable by passing `strings.NewReader`
- Reads all from `r`, unmarshals as `PreToolUseHookInput`, calls `client.SendAuditEvent`
- Must pass `grpc.WaitForReady(true)` to `SendAuditEvent` (matches current behavior at `hook_audit.go:76`)
- Returns `*handler.HandlerError{}` on success (allow), regular `fmt.Errorf` on failure
- Preserves exact error and call semantics of the current code (except the intentional improvement to `io.Copy` error propagation noted in step 1)

### 3. Slim down `cmd/crabswarm/commands/hook_audit.go` — thin cobra wrapper

The command becomes a ~20-line wrapper responsible only for:

- `reader := stdiopipe.Stdin(ctx)` followed by `defer reader.Close()`
- `resolveSocketPath(cmd)` + `os.MkdirAll`
- `grpc.NewClient(...)` + `defer conn.Close()` + `pb.NewAuditServiceClient(conn)`
- `return crabswarm.HookAudit(ctx, reader, client)`

### 4. Add `pkg/crabswarm/hook_test.go` — unit tests

Mock `AuditServiceClient` (single-method interface, trivial to mock). Test cases:

- **Valid input** → `HandlerError` returned, `SendAuditEvent` called with correct `toolName` and non-nil timestamp
- **Invalid JSON** → regular error (not `HandlerError`)
- **Read error** → regular error (not `HandlerError`)
- **Send error** → regular error (not `HandlerError`)

Use a single JSON object (one line from `hook_input.jsonl`) as test fixture — the full file is multi-line JSONL and `io.ReadAll` + `protojson.Unmarshal` expects a single object.

Note: `grpc.WaitForReady(true)` is passed internally by `HookAudit`, not by the caller. Testing call options on a mock requires capturing `...grpc.CallOption` in the mock's `SendAuditEvent`. Add a test that verifies `opts` is non-empty (the mock captures and asserts `len(opts) > 0`).

### 5. Write `pkg/crabswarm/server.go` — `Server` struct scaffold

```go
type Server struct{}

func NewServer() *Server
```

Add `NewServer` constructor. The struct is empty for now — future work will move serve logic here.

## Files to modify

| File                                   | Action                               |
| -------------------------------------- | ------------------------------------ |
| `cmd/internal/stdiopipe/stdiopipe.go`  | Create — `Stdin`                   |
| `pkg/crabswarm/hook.go`               | Write — `HookAudit` function           |
| `pkg/crabswarm/server.go`             | Write — `Server` struct + `NewServer`  |
| `pkg/crabswarm/hook_test.go`          | Create — unit tests with mock client   |
| `cmd/crabswarm/commands/hook_audit.go` | Modify — slim to thin wrapper          |

## Key existing code to reuse

- `pb.AuditServiceClient` interface — `pkg/api/gen/proto/go/claude_hook/v1/audit_service_grpc.pb.go:30`
- `handler.HandlerError` — `pkg/claudehook/handler/handler.go:44`
- `resolveSocketPath` — stays in `cmd/crabswarm/commands/sockpath.go`

## Verification

1. `go build ./...` — compiles
2. `go test ./pkg/crabswarm/...` — new tests pass
3. `go test ./...` — all existing tests still pass
4. `go vet ./...` — no issues
