# Status

Current state: all 9 steps implemented, and the fixes from the review of
the branch are applied — the bridge re-attends after the daemon forgets a
member, the Join RPC is bounded per attempt, an unchanged state report no
longer publishes an event, and the comments and docs that still described
the removed session-start join or a done-only nudge gate are corrected.

Next action: user review.

## Checklist

- [x] 1. Proto `WatchRoom` RPC + regenerate (D4: "server-streaming `ChatService.WatchRoom`")
- [x] 2. Daemon per-room event fan-out + `WatchRoom` impl (D4) — disconnect-slowest policy; stream token interceptor and bounded GracefulStop added as required enablers
- [x] 3. Staleness fallback in nudge gate, `state_reported_at` column (D6: "staleness threshold … in the `notify.SendKeys` gate") — threshold is an unexported 10min const; old dev DBs need deleting (no migration, per no-backcompat rule)
- [x] 4. Hook split `Notification[idle_prompt] → done` (D6: "splitting the current catch-all `Notification → waiting`") — codex cannot see Notification events at all, so codex relies on the staleness fallback (step 3) alone
- [x] 5. Bridge package `crabswarm/chat/mcpserver`, auto-Join + 4 tools (D1, D2, D3) — SDK v1.7.0; join is async with per-tool ensureJoined; `chat_members` mirrors the CLI renderer byte-for-byte, which today prints team/name only (proto `Member` has no state field — step 6 adds it for the resource)
- [x] 6. `crabswarm://chat/members` resource + `resources/updated` (D4); history resource blocked on plan 05 — `Member.state` added to the proto (D8) so ListMembers/events carry state; CLI output unchanged
- [x] 7. CLI entry `crabswarm chat mcp` (thin, per repo rule) — verified live over stdio (initialize/tools_list, stderr-only logs, exit 0 on SIGTERM)
- [x] 8. apm-package MCP registration, SessionStart join removed (D5: "the `SessionStart → chat join` entry is removed", user decision) — self-defined stdio MCP in apm.yml renders into .mcp.json and .codex/config.toml; transitive installs need --trust-transitive-mcp (documented in README)
- [x] 9. e2e: dual-bridge idempotent join, tool round-trip, interrupt heal — heal path was already covered by the hook e2e; the post-staleness nudge leg is unreachable without exporting the 10min const (skipped)

## Blocked / external

- History resource (step 6 tail) and `MessageAppended` events: blocked on
  plan 2026-08-31-05-per_room_message_history.
