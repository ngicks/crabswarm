---
tags: chat mcp join cmdman
---

# The MCP bridge never re-joins on its own after the ~3s startup window (2026-09-02)

`crabswarm chat mcp` has three join mechanisms and each has a hole
(`crabswarm/chat/mcpserver/mcpserver.go`, `resources.go`):
`joinWithRetry` makes 5 attempts (200→1600 ms, ≈3.2 s total) and then
the goroutine exits for the life of the process; `ensureJoined` re-joins
lazily but only when the model invokes a chat tool or resource, and
needs two calls after a reap (the first fails `Unauthenticated` and
merely clears the cached membership via `forgetJoined`); `watchMembers`
is the only infinite retry loop but blocks on `watchWanted`, closed only
by an MCP `resources/subscribe` that Claude Code never sends. The gRPC
transport reconnects on its own; the *membership* does not. The hooks
cannot heal it either: `report-state` does not join, so after a daemon
restart, a recreated `chat.db`, or a bridge that started before
`crabswarm serve` was listening, every `report-state` fails
`Unauthenticated "token is not attending any room; join first"`,
`hook exec` swallows it, and the cmdman switcher shows nothing — which
is exactly "status never changes in `cmdman tui`". A re-join would heal
the display: `Service.Join` on an existing member re-publishes the
stored state through `CmdmanStatusMirror` (`crabswarm/chat/status.go`),
so the missing piece is a trigger, not a mechanism. This was designed
as "retry with backoff; failure degrades to erroring tools"
(`doc/plan/2026-08-31-02-chat_mcp_server/PLAN.md`), which assumed tool
calls would re-join.

Follow-up: make the bridge keep its membership for its whole lifetime —
either turn `joinWithRetry` into a session-long loop with capped
backoff, or (better) start `watchMembers` unconditionally instead of
gating it on a subscription, so a dropped stream is itself the
reconnect signal and a reap is noticed within one backoff regardless of
what the harness subscribes to. Add an e2e: bridge started before the
daemon, then daemon comes up; and daemon restarted under a live bridge.
