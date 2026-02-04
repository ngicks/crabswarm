# Claude Code Permission Hook Scaffold Plan (gRPC + buf)

**Date**: 2026-02-03
**Status**: Ready for Review

## Overview

Go server-client system for Claude Code hook permission handling using gRPC:
- **Single binary** with cobra subcommands
- **Root command** (no subcommand) = Hook client (called by Claude Code)
- **`serve` subcommand** = Interactive permission server
- **Build**: buf for protobuf code generation

## Architecture

```
Claude Code ──stdin(JSON)──► crabhook ──gRPC (Unix socket)──► crabhook serve
             ◄──stdout(JSON)──        ◄──────────────────────  (interactive terminal)
```

## Project Structure

```
crabswarm/
├── buf.yaml                              # Buf module config
├── buf.gen.yaml                          # Buf code generation config
├── go.mod
│
└── hook/                                 # All hook-related code
    ├── api/
    │   ├── scheme/
    │   │   └── proto/                    # Protocol definitions
    │   │       └── permission/v1/permission.proto
    │   │
    │   ├── gen/
    │   │   └── go/                       # Generated code (per language)
    │   │       └── permission/v1/
    │   │           ├── permission.pb.go
    │   │           └── permission_grpc.pb.go
    │   │
    │   └── impl/
    │       └── go/                       # Implementation (per language)
    │           └── permission/v1/service.go
    │
    ├── cmd/
    │   └── crabhook/
    │       ├── main.go                   # Entry point (only calls root.Execute())
    │       └── internal/
    │           ├── root.go               # Root command (client/hook mode)
    │           └── serve.go              # Serve subcommand (server mode)
    │
    ├── model/
    │   └── types.go                      # Claude Code hook I/O types (public)
    │
    └── internal/
        └── server/
            ├── server.go                 # gRPC server + request queue
            └── prompt.go                 # Terminal prompt
```

## CLI Design (cobra)

```bash
# Client mode (root command) - called by Claude Code hook
crabhook                    # Reads stdin, connects to server, returns decision

# Server mode - user runs in terminal
crabhook serve              # Start interactive permission server
crabhook serve --socket /path/to/socket.sock  # Custom socket path
```

**Root command behavior:**
- Read JSON from stdin (Claude Code hook input)
- Connect to gRPC server via Unix socket
- Send permission request
- Wait for response
- Write JSON to stdout (Claude Code hook output)
- Exit 0 on success, exit 2 on error (with fallback to "ask")

**Serve command behavior:**
- Start gRPC server on Unix socket
- Accept connections
- Display incoming requests in terminal
- Prompt user for decision [1] Allow [2] Deny [3] Ask
- Send response to client

## buf Configuration

**buf.yaml:**
```yaml
version: v2
modules:
  - path: hook/api/scheme/proto
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

**buf.gen.yaml:**
```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/ngicks/crabswarm/hook/api/gen/go
plugins:
  - local: protoc-gen-go
    out: hook/api/gen/go
    opt: paths=source_relative
  - local: protoc-gen-go-grpc
    out: hook/api/gen/go
    opt: paths=source_relative
```

**Prerequisite tools:** `protoc-gen-go`, `protoc-gen-go-grpc`

## gRPC Service Definition

**hook/api/scheme/proto/permission/v1/permission.proto:**
```protobuf
syntax = "proto3";

package permission.v1;

option go_package = "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1;permissionv1";

service PermissionService {
  // Request permission for a tool call
  rpc RequestPermission(PermissionRequest) returns (PermissionResponse);
}

message PermissionRequest {
  string request_id = 1;
  string session_id = 2;
  string tool_name = 3;
  bytes tool_input_json = 4;  // JSON bytes
  string cwd = 5;
}

message PermissionResponse {
  string request_id = 1;
  Decision decision = 2;
  string reason = 3;
}

enum Decision {
  DECISION_UNSPECIFIED = 0;
  DECISION_ALLOW = 1;
  DECISION_DENY = 2;
  DECISION_ASK = 3;  // Defer to built-in dialog
}
```

## Implementation Steps

### Phase 1: Protocol Definition
1. Create `buf.yaml` and `buf.gen.yaml` for buf configuration
2. Create `hook/api/scheme/proto/permission/v1/permission.proto`
3. Generate Go code with `buf generate`

### Phase 2: Claude Code Types
4. Create `hook/model/types.go` - Claude Code hook JSON input/output

### Phase 3: gRPC Implementation
5. Create `hook/api/impl/go/permission/v1/service.go` - gRPC service implementation
6. Create `hook/internal/server/server.go` - Server wiring + request queue
7. Create `hook/internal/server/prompt.go` - Terminal prompt and selection

### Phase 4: CLI Commands (cobra)
8. Create `hook/cmd/crabhook/main.go` - Entry point (only calls root.Execute())
9. Create `hook/cmd/crabhook/internal/root.go` - Root command (client mode, uses generated gRPC client)
10. Create `hook/cmd/crabhook/internal/serve.go` - Serve subcommand (server mode)

## Key Dependencies

```go
require (
    github.com/spf13/cobra v1.x.x
    google.golang.org/grpc v1.x.x
    google.golang.org/protobuf v1.x.x
)
```

**Build tools:**
- `buf` - Modern Protocol Buffers toolchain (https://buf.build)

## Key Design Decisions

1. **Single binary** - Simplifies distribution, `hook` for client, `hook serve` for server
2. **Unix socket for gRPC** - `unix:///tmp/crabswarm.sock` for local-only communication
3. **Unary RPC** - Simple request/response pattern
4. **Channel-based request queue** - Single prompt goroutine, FIFO order
5. **Fallback to "ask"** on any error

## Concurrency Model

```go
// hook/api/impl/go/permission/v1/service.go

type Service struct {
    permissionv1.UnimplementedPermissionServiceServer
    requests chan *PendingRequest
}

type PendingRequest struct {
    Ctx      context.Context
    Request  *permissionv1.PermissionRequest
    Response chan *permissionv1.PermissionResponse
}

// gRPC handler - queues request and waits for response
func (s *Service) RequestPermission(ctx context.Context, req *permissionv1.PermissionRequest) (*permissionv1.PermissionResponse, error) {
    respCh := make(chan *permissionv1.PermissionResponse, 1)

    // Non-blocking enqueue with context check
    select {
    case s.requests <- &PendingRequest{Ctx: ctx, Request: req, Response: respCh}:
    case <-ctx.Done():
        return nil, status.Error(codes.DeadlineExceeded, "queue timeout")
    }

    select {
    case resp := <-respCh:
        return resp, nil
    case <-ctx.Done():
        return nil, status.Error(codes.DeadlineExceeded, "decision timeout")
    }
}
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Server not running | Return "ask" (exit 0) |
| Connection timeout | Return "ask" (exit 0) |
| gRPC error | Return "ask" (exit 0) |
| Parse error | Log to stderr, exit 2 |
| Ctrl+C on server | Graceful shutdown |

## Hook Registration

Add to `.claude/settings.json`:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/crabhook",
            "timeout": 300,
            "statusMessage": "Waiting for permission..."
          }
        ]
      }
    ]
  }
}
```

## Verification Plan

1. Generate protobuf code: `buf generate`
2. Build binary: `go build -o bin/crabhook ./hook/cmd/crabhook`
3. Start server: `./bin/crabhook serve`
4. In another terminal, run Claude Code with hook registered
5. Trigger tool call
6. Select option in server terminal
7. Verify Claude Code proceeds

## Build Commands

```bash
# Generate protobuf/gRPC code
buf generate

# Build binary
go build -o bin/crabhook ./hook/cmd/crabhook
```

## Critical Files to Create

1. `buf.yaml` - Buf module configuration
2. `buf.gen.yaml` - Buf code generation config
3. `hook/api/scheme/proto/permission/v1/permission.proto` - gRPC service definition
4. `hook/model/types.go` - Claude Code hook I/O types
5. `hook/api/impl/go/permission/v1/service.go` - gRPC service implementation
6. `hook/internal/server/server.go` - Server wiring + request queue
7. `hook/internal/server/prompt.go` - Terminal prompt
8. `hook/cmd/crabhook/main.go` - Entry point (only calls root.Execute())
9. `hook/cmd/crabhook/internal/root.go` - Root command (uses generated gRPC client)
10. `hook/cmd/crabhook/internal/serve.go` - Serve subcommand (server mode)

---

## Codex Review Feedback (2026-02-03)

### High Priority Fixes

1. **Queue deadlock prevention**
   - Buffer the `requests` channel
   - Use `select` with `ctx.Done()` when enqueueing
   - Prevents blocking if `promptLoop` stalls

2. **Context cancellation handling**
   - Store `ctx` in `PendingRequest`
   - `promptLoop` should skip if `ctx.Done()` already fired
   - Prevents hanging past Claude's hook timeout

3. **Unix socket lifecycle**
   - Remove stale socket on startup (`os.Remove`)
   - Set restrictive permissions (`0600`)
   - Ensure parent directory exists

### Medium Priority

4. **Error semantics**
   - Return proper gRPC status errors for transport issues
   - Client maps errors to "ASK" decision
   - Clearer error handling

5. **Protobuf improvements**
   - Use `bytes tool_input_json` instead of `string`
   - Rename `id` to `request_id` for clarity
   - Add comments for `DECISION_ASK` semantics

### Go Best Practices

- Use `context.Context` throughout for timeout/cancellation propagation
- Keep stdout strictly for hook response; logs to stderr or file only
- Handle SIGTERM/SIGINT with best-effort "ask" response before exit
- Use `insecure.NewCredentials()` for Unix sockets
- Custom `WithContextDialer` for Unix socket client
