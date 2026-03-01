# Review Result: Changes Requested

## Findings

1. High: `server start` does not explicitly guarantee `--sock` is passed to the spawned `serve` process.
- Reference: `PLAN.md` lines 81-90.
- Risk: `server start --sock <custom>` may probe one lock path but launch `serve` on the default socket, causing mismatched lock/socket behavior.
- Required fix: In the plan, explicitly define the exact spawned command as `<resolved-binary> serve --sock <resolved-sock-path>` (properly shell-quoted).

2. High: `tmux.New` semantic change can attach to an existing session with different startup intent.
- Reference: `PLAN.md` lines 45-60 and 85-90.
- Risk: With `DisallowReuse=false`, `New` returning `Attach` may silently ignore new `StartupKeys`, which can hide configuration mismatch and make behavior non-obvious at call sites.
- Required fix: Document this contract explicitly in the plan and add a test that proves reused session behavior does not re-run startup keys.

3. Medium: `syscall.Flock` in shared code is not portable and can break non-Unix builds.
- Reference: `PLAN.md` lines 28-31.
- Risk: `syscall.Flock` is unavailable on some targets; current verification (`go build ./cmd/crabswarm`) does not catch cross-platform regressions.
- Required fix: Specify OS scope (Unix-only) in plan, or use a portability strategy (e.g., Unix-gated implementation + build tags / x/sys/unix) and add at least one build-target check accordingly.

4. Medium: `server start` pre-check has TOCTOU race and no post-start validation.
- Reference: `PLAN.md` line 83.
- Risk: Another process can acquire lock between pre-check and actual server startup; command may report started while server exits immediately.
- Required fix: Add post-start confirmation (e.g., short retry loop checking lock/socket availability or health probe) before printing success.

## Additional Gaps

1. Verification section does not include tests for lock behavior in `pkg/crabswarm/server.go`.
- Reference: `PLAN.md` lines 110-117.
- Suggestion: add focused tests for lock acquisition failure/success paths and stale socket cleanup interaction.

2. Command UX compatibility is unspecified.
- Reference: `PLAN.md` lines 69-77.
- Suggestion: clarify whether existing `serve` remains supported at root and whether `server start` is additive or part of a migration.

## Verdict

Plan is directionally good, but the issues above should be addressed before implementation to avoid behavioral regressions and ambiguous runtime behavior.
