# Chat MCP server (keeping the CLI)

Per-agent stdio MCP server `crabswarm chat mcp` bridging to the central
daemon: auto-join at startup, send/read/members as tools, member state and
room history as subscribable resources; hook-driven state reporting stays,
hardened against ESC interrupts.

## Goal / success criteria

- A harness configured with `crabswarm chat mcp` is a joined room member
  before its first turn, with no `SessionStart → chat join` hook needed.
- The agent can send, broadcast, read its inbox and list members through
  MCP tools; results match the CLI verbs byte-for-byte in content.
- `crabswarm://chat/members` is a readable, subscribable resource that
  fires `resources/updated` on member state changes.
- An ESC-interrupted Claude Code session returns to `done` within ~90s
  and receives nudges again (e2e-verifiable via state + notify gate).
- Every existing CLI verb behaves unchanged.

## Non-goals

- Replacing keystroke-injection nudges: MCP cannot push into an idle
  harness (`doc/plan/2026-08-26-01-chat_subcommand/notification_mechanisms.md`),
  so `notify.SendKeys` stays the mid-idle path.
- MCP-side state reporting (`report_state` is deliberately not a tool —
  see D3).
- The admin plane (plan `2026-08-31-01-chat_admin_subcommand`) and the
  history table itself (plan `2026-08-31-05-per_room_message_history`).

## Context

- Daemon: single gRPC server on a Unix socket
  (`crabswarm/server/server.go:108-178`), serving `ChatService` +
  `ChatAdminService` (`api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto`).
- Member CLI verbs resolve token via `chatcli.ResolveToken`
  (`crabswarm/chat/cli/token.go`) and dial via `chatcli.Dial`
  (`cmd/crabswarm/commands/zz_chat.go:16-42`).
- `Service.Join` is idempotent for a known token
  (`crabswarm/chat/service_member.go:44-53`).
- State reporting: hooks in
  `apm-package/crabswarm-chat/.apm/hooks/report-state.json`; today all
  `Notification` events map to `waiting`, and only `Stop` reaches `done`.
- Nudge gate: `crabswarm/chat/notify/notify.go:58-81` declines unless
  `State == StateDone`.

## Approach

A thin bridge process, not a second daemon: `crabswarm chat mcp` (cobra
entry under `cmd/crabswarm/commands/`, logic in a new
`crabswarm/chat/mcpserver` package per the thin-entrypoint rule) speaks
MCP over stdio using the official Go SDK and forwards every operation to
the existing gRPC `ChatService` with the token resolved once at startup.
The daemon stays the single source of truth; no chat logic moves.

For `resources/updated`, the bridge needs to learn about daemon-side
changes. Chosen: a new server-streaming RPC `WatchRoom` on `ChatService`
(events: member state changed, member joined/left, message appended).
Rejected: bridge-side polling (latency, N bridges hammering SQLite) and
SQLite change hooks in the bridge (bridges must not open the DB; the
daemon owns it). `WatchRoom` is also what the admin TUI plan
(`2026-08-31-06-admin_tui`) needs, so it pays twice.

The ESC-interrupt fix is hook + gate hardening, independent of MCP but
owned here because the issue entry records it here: map
`Notification[idle_prompt]` to `report-state done` (splitting the current
catch-all `Notification → waiting` so only permission-type notifications
mean `waiting`), and add a staleness fallback to the nudge gate.

## Public surface delta

```go
// crabswarm/chat/mcpserver (new package)
package mcpserver

// Server bridges one agent's MCP stdio session to the chat daemon.
type Server struct{ /* unexported */ }

// New dials sockPath, resolves identity for token, and prepares the MCP
// server. Join happens in Run so failures surface as MCP tool errors,
// not a dead harness.
func New(logger *slog.Logger, sockPath, token string) (*Server, error)

// Run serves MCP on stdin/stdout until ctx is done.
func (s *Server) Run(ctx context.Context) error
```

```proto
// api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto
service ChatService {
  // ... existing rpcs unchanged ...
  // WatchRoom streams events of the caller's room until cancelled.
  rpc WatchRoom(WatchRoomRequest) returns (stream RoomEvent);
}
message WatchRoomRequest {}
message RoomEvent {
  oneof event {
    MemberStateChanged member_state_changed = 1;
    MemberJoined       member_joined        = 2;
    MemberLeft         member_left          = 3;
    MessageAppended    message_appended     = 4; // fed by plan 05's room log
  }
}
```

```text
# CLI
crabswarm chat mcp            # run the per-agent stdio MCP server
    --token / $CRABSWARM_CHAT_TOKEN / $CMDMAN_CMD_ID   # as every member verb
    --sock, --config                                    # as every chat verb

# MCP tools (served to the harness)
chat_send      {to: string, message: string}
chat_broadcast {message: string}
chat_read      {}                # drains the inbox, like `chat read`
chat_members   {}                # room members with team/name/state

# MCP resources
crabswarm://chat/members         # caller's room roster + states; subscribable
crabswarm://chat/history         # per-room history; lands with plan 05

# Hook wiring (apm-package/crabswarm-chat/.apm/hooks/report-state.json)
Notification[idle_prompt]      -> crabswarm chat report-state done   # new
Notification[permission_...]   -> crabswarm chat report-state waiting # was: all Notifications
SessionStart -> chat join       # kept for harnesses without MCP configured
```

Durable state vocabulary: none added here (the room log table belongs to
plan 05). No config keys added; the bridge reuses `sock` and the `chat`
block as-is.

## Implementation steps

1. **Proto: `WatchRoom`** — add the RPC + messages above to
   `chat_service.proto`, regenerate (`go generate ./api`). Verifiable:
   generated code compiles, `buf lint` passes.
2. **Daemon: event fan-out** — in `crabswarm/chat/service.go`, publish
   member-state/join/leave events at their mutation points
   (`Service.Join`, `ReportState`, `Leave`, admin `MoveMember`) to a
   per-room broadcaster; implement `WatchRoom` filtering by the caller's
   room. Verifiable: unit test subscribing then mutating sees the events.
3. **Staleness fallback in the nudge gate** — `notify.SendKeys.Notify`
   (`crabswarm/chat/notify/notify.go:64-71`): a `working`/`waiting`
   state older than a threshold (default 10min, recorded via a
   `state_reported_at` column alongside `members.state`) no longer
   blocks the nudge; the dialog-marker capture-screen check remains the
   safety net. Verifiable: unit test with a stale state.
4. **Hook split: `idle_prompt` → done** — rework
   `apm-package/crabswarm-chat/.apm/hooks/report-state.json` so
   `Notification` matches `idle_prompt` → `report-state done` and
   permission-type notifications → `waiting`; mirror for the codex
   target if its notify supports the distinction. Verifiable: e2e hook
   test (`e2e/crabswarm/chat_hooks_test.go`) simulating interrupt then
   idle notification sees state `done`.
5. **Bridge package `crabswarm/chat/mcpserver`** — official Go MCP SDK
   (`github.com/modelcontextprotocol/go-sdk`); startup: resolve token,
   dial, `Join` (retry with backoff; failure degrades to erroring
   tools); serve the four tools by delegating to the same
   `chatv1.ChatServiceClient` calls the CLI client makes
   (`crabswarm/chat/cli/member.go`). Verifiable: unit tests against the
   in-process stub server used by `crabswarm/chat/cli/client_test.go`.
6. **Resources + subscriptions** — serve `crabswarm://chat/members` from
   `ListMembers`; on `WatchRoom` events, emit `resources/updated` to
   subscribed sessions. `crabswarm://chat/history` registers only when
   plan 05's read RPC exists (boundary ledger). Verifiable: unit test —
   subscribe, flip a member state, observe the notification.
7. **CLI entry `chat mcp`** — `cmd/crabswarm/commands/chat_mcp.go`,
   flags-only, hands off to `mcpserver.New(...).Run(ctx)`. Verifiable:
   `crabswarm chat mcp` under a live daemon answers `initialize` and
   `tools/list` over stdio.
8. **apm package wiring** — add the MCP server registration to
   `apm-package/crabswarm-chat` for claude and codex targets; drop
   nothing (SessionStart join stays, D5). Verifiable: e2e chat test
   variant where join comes only from the bridge.
9. **e2e** — extend `e2e/crabswarm/chat_test.go`: two bridges in one
   container-shape (same token) both start cleanly; send/read via tools
   round-trips; interrupted-state heal path from step 4.

## Testing and verification

Unit tests beside each package (steps 2, 3, 5, 6); hook-level e2e for the
idle_prompt split (step 4); full-path e2e (step 9). `go build ./...`,
`go test ./...`, `buf lint` after step 1.

## Boundary ledger

| Deliverable | Owner |
| --- | --- |
| `WatchRoom` RPC + daemon fan-out | this plan, steps 1–2 |
| MCP bridge, tools, members resource | this plan, steps 5–7 |
| idle_prompt hook split + staleness fallback | this plan, steps 3–4 |
| Room history table + read RPC | plan 2026-08-31-05-per_room_message_history |
| `crabswarm://chat/history` resource registration | this plan, step 6 (blocked on 05) |
| `MessageAppended` event emission on send | plan 05 (writes the log) feeding step 2's broadcaster |
| Admin plane / TUI consumption of `WatchRoom` | plans 2026-08-31-01, 2026-08-31-06 |

## Risks

- Go MCP SDK stdio lifecycle vs. harness process management: a harness
  that never sends `initialize` leaves a bridge holding a `WatchRoom`
  stream; scope streams to the MCP session lifetime.
- `idle_prompt` matcher semantics (~60s delay) are Claude-Code-specific;
  codex may lack an equivalent, leaving codex members reliant on the
  staleness fallback alone.
- gRPC streaming over the shared Unix socket multiplies open streams per
  agent; bound with per-room broadcaster buffer + drop-slowest policy.

## Open questions

(none — all resolved automatically; see DECISION.md D1–D6)
