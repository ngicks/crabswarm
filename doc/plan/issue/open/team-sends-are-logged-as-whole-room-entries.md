---
tags: chat admin history proto store
---

# Team sends are logged as whole-room entries (2026-09-03)

`AdminSendRequest.target` has a team case, and `broadcastTeamFrom`
(`crabswarm/chat/inbox.go`) delivers it to every current member of the
team — but it records the message with no recipient, because the
history reader (`crabswarm/chat/history.go`, the branch that only
rebuilds a recipient when a member name is present) would drop a
team-only recipient on read. `crabswarm chat admin history`, the TUI's
conversation pane and `render.go`'s transcript therefore show a team
send as `admin → *`, indistinguishable from a whole-room send.

Follow-up: store a team-only recipient (a `to_team` without a name) and
teach the read side to render it, so the entry says `admin → beta/*`.
Touches the `room_log` schema, the sqlc queries and `AdminHistoryEntry`.
