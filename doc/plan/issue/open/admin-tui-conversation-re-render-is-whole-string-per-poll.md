---
tags: chat admin tui perf
---

# Admin TUI conversation re-render is whole-string per poll (2026-09-02)

`layout()` (`crabswarm/chat/cli/tui/model.go`) re-renders the entire
conversation string (bounded at 2000 entries) on every poll that brings
entries. Fine at terminal scale.

Follow-up: make it incremental only if the screen ever feels heavy.
Related: `model.go` is ~395 LoC against the repo's 300-LoC preference;
splitting it is a natural companion cleanup.
