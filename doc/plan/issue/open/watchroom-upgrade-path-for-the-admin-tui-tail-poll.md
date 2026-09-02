---
tags: chat admin tui proto
---

# WatchRoom upgrade path for the admin TUI tail poll (2026-09-02)

The TUI tails the room log by cursor poll (~1s) against
`ChatAdminService.History`. The daemon already serves server-streaming
`ChatService.WatchRoom` on the member plane; once an admin-plane stream
(or a message-appended event feeding one) exists, the poll can be
replaced without changing the `tui` package's `LogReader` consumers.

Follow-up: revisit after the MessageAppended producer decision above.
