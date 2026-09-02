---
tags: chat join reaping
---

# Flagless chat members are never reaped and hold their names forever (2026-09-02)

`checkLiveness` (`crabswarm/chat/service.go`) is deliberately
agent-only, so a `KindHuman` member is never reaped even when the
provider forgets its token — it drops out of the cmdman status display
but holds its name against colliding joiners indefinitely. Intended for
admin-registered humans; an explicit `chat leave` is the current answer.

Follow-up: revisit if plain-shell membership becomes common.
