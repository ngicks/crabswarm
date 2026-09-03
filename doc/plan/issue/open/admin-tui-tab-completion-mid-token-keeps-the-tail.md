---
tags: chat admin tui completion
---

# Admin TUI Tab completion mid-token keeps the tail (2026-09-03)

`token()` in `crabswarm/chat/cli/tui/completion.go` reads only the text
left of the cursor, and `replaceToken` backspaces exactly that many
runes, so Tab on `@alx` with the cursor after `@al` yields
`@backend/alice x`. Shell-like, but every completion test tabs at the
end of the buffer, so nothing pins it either way.

Follow-up: decide whether the rest of the word should be replaced too,
and add the mid-token case to `completion_test.go`.
