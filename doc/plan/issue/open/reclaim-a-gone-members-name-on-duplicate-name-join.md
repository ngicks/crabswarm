---
tags: chat join cmdman store
---

# Reclaim a gone member's name on duplicate-name join (2026-08-31)

A recreated compose replica cannot rejoin the chat room: the identity
token is `$CMDMAN_CMD_ID`, the per-instance cmdman command ID
(`crabswarm/chat/cli/token.go`), and cmdman compose recreate (also
`down`+`up` and `scale` down/up) is remove-then-create — the replica
comes back with a new command ID but the same `cmdman.compose.command` /
`cmdman.compose.scale-index` labels, so it derives the same default name
(e.g. `worker-1`). The new token is unknown to the store, `Service.Join`
takes the fresh-join path, and `Store.Join` rejects the taken name
(`crabswarm/chat/member.go` → `codes.AlreadyExists`). The stale member
holding the name is never freed: reaping is lazy and fires only when the
stale member itself makes an RPC (`crabswarm/chat/service.go`), its
command is gone so it never will, there is no sweep or admin
member-removal RPC, and the store persists across daemon restarts
(`$XDG_STATE_HOME/crabswarm/chat.db`). The only workaround is an
explicit `--name`, which defeats the label-derived-naming feature.

Decided policy: on a duplicate-name join, the server checks whether the
existing holder of that name is still around — resolve its stored token
through the team-info provider — and when the previous member is gone
(token no longer resolves), frees the name and admits the joiner;
a collision with a live member stays a clear `AlreadyExists` rejection.

Regression test to add with the fix: `Service.Join` with a
provider-derived name already taken in the same team by a token the
provider no longer knows.
