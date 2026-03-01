# Plan Review: Add `server start` Subcommand

## Findings

1. **Medium** — Missing automated verification for new command behavior
   - Reference: `PLAN.md:57-62`
   - The verification section only includes `go build`, `go vet`, and manual checks. This does not protect against regressions in:
     - command wiring (`root -> server -> start`),
     - startup command construction (quoting/path handling),
     - expected handling of `mux.ErrSessionExists`.
   - Suggested fix: add at least one automated test in `cmd/crabswarm/commands` (or a small extraction unit test) that validates startup command generation and error-branch behavior.

## Summary

Plan is directionally sound and implementation choices align with existing code patterns, but test coverage should be part of the plan before implementation is considered complete.
