# Status

Current state: in progress — step 1 done (proto `WatchRoom` + regen,
`buf lint`/`go build` green; `RoomEvent` name kept via per-RPC lint
ignore, see D7).

Next action: step 2 (daemon fan-out + `WatchRoom` impl).

## Checklist

- [x] 1. Proto `WatchRoom` RPC + regenerate (D4: "server-streaming `ChatService.WatchRoom`")
- [x] 2. Daemon per-room event fan-out + `WatchRoom` impl (D4) — disconnect-slowest policy; stream token interceptor and bounded GracefulStop added as required enablers
- [x] 3. Staleness fallback in nudge gate, `state_reported_at` column (D6: "staleness threshold … in the `notify.SendKeys` gate") — threshold is an unexported 10min const; old dev DBs need deleting (no migration, per no-backcompat rule)
- [x] 4. Hook split `Notification[idle_prompt] → done` (D6: "splitting the current catch-all `Notification → waiting`") — codex cannot see Notification events at all, so codex relies on the staleness fallback (step 3) alone
- [ ] 5. Bridge package `crabswarm/chat/mcpserver`, auto-Join + 4 tools (D1, D2, D3)
- [ ] 6. `crabswarm://chat/members` resource + `resources/updated` (D4); history resource blocked on plan 05
- [ ] 7. CLI entry `crabswarm chat mcp` (thin, per repo rule)
- [ ] 8. apm-package MCP registration, SessionStart join removed (D5: "the `SessionStart → chat join` entry is removed", user decision)
- [ ] 9. e2e: dual-bridge idempotent join, tool round-trip, interrupt heal

## Blocked / external

- History resource (step 6 tail) and `MessageAppended` events: blocked on
  plan 2026-08-31-05-per_room_message_history.
