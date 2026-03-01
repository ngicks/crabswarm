# Plan Review: Add `server start` Subcommand + `DisallowReuse`

## Findings

Ignore 1. since there's not real consumer for this pkg.

<!---->
<!-- 1. **High** — `tmux.New` zero-value behavior change is a broad API break -->
<!--    - Reference: `PLAN.md:7`, `PLAN.md:20-21`, `PLAN.md:50` -->
<!--    - The plan changes `tmux.New` from "create-new-or-error" to "create-or-attach" by default. This affects every current and future caller that relies on duplicate detection semantics, not just `server start`. -->
<!--    - Risk: hidden regressions outside the currently listed tests/call-sites, because callers may silently attach to an existing session and proceed under wrong assumptions. -->
<!--    - Suggested fix: either (a) keep existing default semantics and use an explicit opt-in flag for reuse, or (b) if the default must change, include a full call-site audit and explicit migration notes/tests for all known `tmux.New` users. -->

2. **Medium** — Verification misses automated tests for command wiring and user-visible behavior
   - Reference: `PLAN.md:98-104`
   - Verification includes package tests/build/vet and manual checks, but no automated tests for the new command path.
   - Risk: regressions in `root -> server -> start` wiring, startup command composition, and status output can slip in without detection.
   - Suggested fix: add command-level tests in `cmd/crabswarm/commands` for at least:
     - `server start` registration/wiring,
     - successful start path output,
     - reuse path output and exit status,
     - tmux error propagation.

## Summary

<!-- The plan is implementable, but it currently underestimates the compatibility impact of changing `tmux.New` default semantics and lacks automated coverage for the new CLI behavior. -->
