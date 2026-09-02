---
tags: chat admin cli completion
---

# Shell completion for the admin TUI's room argument (2026-09-02)

The admin can already enumerate rooms and the completion precedent
exists (`completeChatMembers`, `cmd/crabswarm/commands/zz_chat.go`), but
`chat admin tui --room` completes nothing.

Follow-up: wire room-name completion for the flag (and for the other
admin verbs' room positional while at it).
