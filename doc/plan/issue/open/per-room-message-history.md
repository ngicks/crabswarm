---
tags: chat history store
---

# Per-room message history (2026-08-31)

Chat history does not exist: the store is a pure inbox — per-recipient
message rows drained with `DELETE FROM messages WHERE recipient = ?` on
read (`crabswarm/chat/internal/schema/ddl/schema.sql`,
`queries/queries.sql`). Nothing retains what was said, so neither the
admin nor a member can look back at old conversation, which would be a
useful resource for agents catching up on context.

Follow-up: add an append-only per-room log table written on send, with
a retention cap (row limit and/or age-based pruning), plus read access
for both the admin plane and members (CLI verb now; an MCP resource
once the server lands).
