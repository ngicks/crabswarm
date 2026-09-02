---
tags: chat join cmdman needs-info
---

# Admin TUI "join failed: missing row" — needs the verbatim error (2026-09-02)

Reported: joining from the admin TUI failed with a missing-row error.
Investigated but unresolved, because the error text and the exact
command are unknown. Ruled out: the store cannot emit
`sql: no rows in result set` — the only two `:one` queries go through
`memberFrom`, which maps `sql.ErrNoRows` to `ErrNotFound`, and
`storeStatus` (`crabswarm/chat/service.go`) turns that into a gRPC
`NotFound` with the store's own wording; there is no `tokens` table
(admin auth is age-nonce, `crabswarm/chat/auth/age.go`, failing as
`Unauthenticated`); the schema is current (see the migrations entry).
The admin TUI itself never joins: `openRoom`
(`crabswarm/chat/cli/tui/tui.go`) only lists rooms and says
`no room %q: the daemon knows <list>`. So "join" must have been
`crabswarm chat join` from a shell, and the ranked `NotFound`
candidates are: `$CMDMAN_CMD_ID` unset or unknown to cmdman
(`no team information for this token: ... cmdman knows no command`,
`crabswarm/chat/service_member.go`); the command exists but is outside
a compose project (`crabswarm/chat/resolver/cmdman.go`); a re-join of
an agent whose cmdman command was recreated (`token is no longer known
to the team-info provider`); or a TUI send to a name that is not in the
room (`member not found`, `crabswarm/chat/admin_rooms.go`).

Follow-up: capture the verbatim error and `echo $CMDMAN_CMD_ID` from
the shell that ran it, then either close this as operator error or
turn the matching candidate into a clearer message. Independently:
every one of those messages could name the token and the room it
looked in, and the TUI's `openRoom` rejection should list what it
tried, so the next report carries its own diagnosis.
