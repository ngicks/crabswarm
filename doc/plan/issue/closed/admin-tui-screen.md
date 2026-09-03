---
tags: chat admin tui
---

# Admin TUI screen (2026-08-31)

The admin needs its own interactive surface: unlike members it always
specifies a room id, and its job is watching over the agents'
conversation rather than participating in an inbox. Today there is no
TUI code anywhere under `crabswarm/chat/` or
`cmd/crabswarm/commands/`, `chat read` is one-shot with no follow mode,
and the `web/` SPA has generated chat proto types
(`web/src/gen/ngicks/crabswarm/chat/v1/`) that nothing consumes.

Follow-up: build an admin TUI (room picker / explicit room id, live
view of the room's conversation via the per-room history plus a
streaming or polling tail, send-as-admin input). Depends on the `chat
admin` subcommand and per-room history entries above.

## Conclusion

Resolved 2026-09-03. `crabswarm chat admin tui` exists under
`crabswarm/chat/cli/tui/`: a rooms pane selects the room (`--room` is an
optional initial selection), the conversation pane tails the per-room
history by polling, the members pane shows the roster, and a multi-line
message pane sends as admin with `@` addressing. Streaming instead of the
tail poll stays open as "WatchRoom upgrade path for the admin TUI tail
poll".
