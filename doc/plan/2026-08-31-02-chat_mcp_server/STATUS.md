# Status

Current state: in progress — step 1 done (proto `WatchRoom` + regen,
`buf lint`/`go build` green; `RoomEvent` name kept via per-RPC lint
ignore, see D7).

Next action: step 2 (daemon fan-out + `WatchRoom` impl).

## Checklist

- [x] 1. Proto `WatchRoom` RPC + regenerate (D4: "server-streaming `ChatService.WatchRoom`")
- [ ] 2. Daemon per-room event fan-out + `WatchRoom` impl (D4)
- [ ] 3. Staleness fallback in nudge gate, `state_reported_at` column (D6: "staleness threshold … in the `notify.SendKeys` gate")
- [ ] 4. Hook split `Notification[idle_prompt] → done` (D6: "splitting the current catch-all `Notification → waiting`")
- [ ] 5. Bridge package `crabswarm/chat/mcpserver`, auto-Join + 4 tools (D1, D2, D3)
- [ ] 6. `crabswarm://chat/members` resource + `resources/updated` (D4); history resource blocked on plan 05
- [ ] 7. CLI entry `crabswarm chat mcp` (thin, per repo rule)
- [ ] 8. apm-package MCP registration, SessionStart join removed (D5: "the `SessionStart → chat join` entry is removed", user decision)
- [ ] 9. e2e: dual-bridge idempotent join, tool round-trip, interrupt heal

## Blocked / external

- History resource (step 6 tail) and `MessageAppended` events: blocked on
  plan 2026-08-31-05-per_room_message_history.
