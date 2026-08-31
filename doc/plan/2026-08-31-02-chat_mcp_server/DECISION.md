# Decisions

## D1 — Bridge process, not a second daemon (automatic decision)

`crabswarm chat mcp` is a thin per-agent stdio bridge forwarding to the
existing gRPC `ChatService`; the daemon stays the single owner of the
SQLite store and all chat logic. Rejected: embedding an MCP listener in
the daemon itself (MCP is per-agent stdio in every harness's config
model; a shared HTTP MCP endpoint would reintroduce per-agent auth that
the token already solves) and a standalone MCP daemon owning the DB (two
writers is what the flock in `server.go:110-120` exists to prevent).

## D2 — Official Go MCP SDK (automatic decision)

Use `github.com/modelcontextprotocol/go-sdk`. Rejected: hand-rolling
JSON-RPC (the spec's lifecycle/subscription surface is large) and
third-party SDKs (the official one tracks the 2026-07-28 spec the plan
is grounded in).

## D3 — No `report_state` MCP tool (automatic decision)

State reporting stays hook → CLI (`crabswarm chat report-state`).
Protocol necessity: MCP has no client→server state push or turn-end
signal, and Claude Code hooks cannot invoke MCP tools
(anthropics/claude-code#26112). Exposing the tool to the model itself
was rejected outright: the model reporting its own harness state is
exactly the unreliable self-report the hook design avoids. Revisit if
#26112 lands.

## D4 — `WatchRoom` streaming RPC over polling (automatic decision)

Resource subscriptions are fed by a new server-streaming
`ChatService.WatchRoom` with daemon-side per-room fan-out. Rejected:
bridge-side polling (latency and N-bridges load) and bridges opening the
SQLite DB directly (daemon owns the store). The RPC doubles as the feed
for the admin TUI plan (2026-08-31-06-admin_tui).

## D5 — Remove the `SessionStart → chat join` hook (user decision, 2026-08-31)

The bridge's startup auto-join is the sole join path; the
`SessionStart → chat join` entry is removed from
`apm-package/crabswarm-chat/.apm/hooks/report-state.json`. Decided by
the user ("(2) => remove start hook"), overriding this entry's earlier
automatic choice to keep it. Consequence: joining now requires the MCP
server configured — a harness wired with only the hooks package no
longer joins at session start (the CLI `chat join` remains for manual
use).
Rejected (previous automatic choice): keeping the hook for
harnesses without MCP configured, on the grounds Join idempotency
makes duplication free.

## D6 — ESC-interrupt heal: idle_prompt hook + staleness fallback (automatic decision)

Two layers: `Notification[idle_prompt] → report-state done` (splitting
the current catch-all `Notification → waiting` in
`apm-package/crabswarm-chat/.apm/hooks/report-state.json`), plus a
staleness threshold (default 10min, via a new `state_reported_at`
column) in the `notify.SendKeys` gate so a missed hook cannot wedge a
member forever. Grounded in: "Stop hooks … don't fire on user
interrupts" (code.claude.com/docs/en/hooks-guide). Rejected: hook-only
(codex has no idle_prompt equivalent) and gate-only (a 10min heal alone
is a poor experience when the ~60s hook signal exists).

## D7 — Keep `RoomEvent` name, suppress lint per-RPC [automatic]

`buf lint`'s `RPC_RESPONSE_STANDARD_NAME` wants the `WatchRoom` stream
element named `WatchRoomResponse`. Kept `RoomEvent` with a per-RPC
`buf:lint:ignore` comment: the event feed outlives this one RPC (the
admin TUI subscribes to the same stream), and a comment ignore is
narrower than an `ignore_only` config entry. Rejected: renaming to
`WatchRoomResponse` (misnames the type everywhere else it is consumed).

## D8 — Member state exposure: proto field + resource, not tool output [automatic]

The plan wants `chat_members` to show team/name/state, but also requires
tool results to match the CLI verbs byte-for-byte, and `chatv1.Member`
carried no state field. Resolution: byte-for-byte wins (it is a success
criterion), so `chat_members` keeps mirroring `cli.RenderMembers`
(team/name lines); a `HarnessState state` field is added to
`chatv1.Member`, filled by the daemon's ListMembers, and the
`crabswarm://chat/members` resource carries the roster including state.
Rejected: diverging the tool output from the CLI renderer.
