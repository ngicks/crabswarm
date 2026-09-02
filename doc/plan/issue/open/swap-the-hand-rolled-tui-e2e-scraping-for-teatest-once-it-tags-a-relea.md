---
tags: tui e2e test
---

# Swap the hand-rolled TUI e2e scraping for teatest once it tags a release (2026-09-02)

`charm.land/x/exp/teatest/v2` resolves as a module path but has no
tagged version, so `e2e/crabswarm/chat_tui_test.go` strips ANSI from
accumulated program output itself.

Follow-up: adopt teatest when it tags a release.
