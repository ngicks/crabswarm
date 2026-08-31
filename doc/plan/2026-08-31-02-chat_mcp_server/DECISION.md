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

## D5 — Keep the `SessionStart → chat join` hook (automatic decision)

The bridge auto-joins at startup, but the hook stays: harnesses without
MCP configured still need it, and `Service.Join` idempotency
(`crabswarm/chat/service_member.go:44-53`) makes the duplication free.
Rejected: removing the hook when the MCP server is present (conditional
hook wiring buys nothing and breaks the no-MCP path).

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
