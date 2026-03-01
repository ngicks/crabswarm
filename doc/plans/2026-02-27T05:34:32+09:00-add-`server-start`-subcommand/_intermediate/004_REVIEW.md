# Plan Review: Add `server start` Subcommand + Config.AllowReuse

## Findings

1. **High** — `server start` can report success while server is not actually running
   - Reference: `PLAN.md:64-73`, `PLAN.md:95-96`
   - The plan makes `server start` always pass `AllowReuse: true`, so any pre-existing tmux session named `crabswarm` is treated as reusable success. If that session is unrelated or stale, no startup keys are sent (as noted), and the server may never be started.
   - Risk: false-positive "started/already running" result with no active gRPC server.
   - Suggested fix: either (a) keep collision as an explicit error for `server start`, or (b) add validation after attach (e.g., probe the expected socket / verify pane command marker) and fail if server is not actually running.

2. **Medium** — Verification still lacks automated coverage for new CLI behavior
   - Reference: `PLAN.md:90-96`
   - Verification includes tmux package tests/build/vet plus manual checks, but no tests for `root -> server -> start` wiring or for startup command construction/error paths.
   - Risk: regressions in command registration, output semantics, and reuse-path behavior can slip through.
   - Suggested fix: add command-level tests in `cmd/crabswarm/commands` for registration/wiring and at least one success + one reuse/collision path.

## Summary

The `AllowReuse` API direction is an improvement over prior default-semantics changes, but the current `server start` reuse behavior is not yet safe enough to guarantee the command’s contract.
