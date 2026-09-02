---
tags: chat admin cli
---

# Admin verb spelling: tui takes --room, the others take a positional (2026-09-02)

Every other room-scoped admin verb (`log`, `send`, ...) takes the room
as its first positional argument; `chat admin tui` requires a `--room`
flag because the plan fixed that spelling. The group is internally
inconsistent.

Follow-up: pick one convention and align the verbs.
