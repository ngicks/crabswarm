# Plan Review: Add `server start` Subcommand + Config.DisallowReuse

## Findings

1. **High** — Reuse-by-name can report success without actually starting crabswarm server
- Reference: `PLAN.md:63-71`, `PLAN.md:93-94`
- The plan makes `server start` call `tmux.New` with default `DisallowReuse: false`, which attaches on duplicate session name. If a pre-existing `crabswarm` tmux session is unrelated/stale, startup keys are not sent and `serve` is never started, but the command still appears successful.
- Impact: false-positive success for a critical lifecycle command.
- Suggested fix: after attach/reuse, verify server liveness (e.g., socket probe/health RPC) and fail if not running; or require explicit non-reuse behavior for `server start`.

2. **Medium** — Verification lacks automated coverage for new CLI behavior
- Reference: `PLAN.md:90-94`
- Verification is mostly manual for `server start` and does not include command-level tests for command wiring and reuse/collision behavior.
- Impact: regressions in registration, command construction, and reuse semantics can pass CI.
- Suggested fix: add tests in `cmd/crabswarm/commands` for `server`/`server start` registration and at least one path each for fresh create and existing-session reuse/collision.

## Summary

The `DisallowReuse` API shape is coherent, but `server start` currently has a correctness gap around name-only session reuse and needs automated CLI-level tests.
