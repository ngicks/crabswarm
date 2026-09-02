---
tags: hook test
---

# Confirm intent: hook path violations now ride the always-JSON output (2026-09-02)

`crabswarm/hook/path/windows.go` calls `handler.Block`, so its violation
report moved with the hook-exec output-template change from exit 2 +
stderr to exit 0 + JSON on stdout, although that feature was scoped to
`hook exec` only. No test, doc, or consumer depends on exit 2 and there
is no deployed consumer, so nothing is broken — but the semantic shift
for `hook path` was never explicitly decided, and the package has no
process-level test at all.

Follow-up: confirm the always-JSON wire form is wanted for `hook path`
too, and pin it with a process-level test either way.
