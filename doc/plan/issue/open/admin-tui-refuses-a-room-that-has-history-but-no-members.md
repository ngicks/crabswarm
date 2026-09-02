---
tags: chat admin tui history
---

# Admin TUI refuses a room that has history but no members (2026-09-02)

`openRoom` (`crabswarm/chat/cli/tui/tui.go`) decides room existence from
the roster listing, so once every member leaves,
`chat admin tui --room R` errors while `chat admin log R` still serves
the retained transcript.

Follow-up: decide existence from the log as well — needs a read the
admin History RPC does not offer today (an "any rows for this room?"
probe or listing rooms present in `room_log`).
