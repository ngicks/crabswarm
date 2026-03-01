# Plan Review: `server start` Subcommand

## Findings

1. **High: Command string is not safely shell-escaped**
- Planned step: build `<binary> serve --sock <path>` and send it via tmux startup keys.
- Risk: if executable path or socket path contains spaces/shell metacharacters, the pane shell will parse it incorrectly (or unexpectedly).
- Recommendation: update the plan to explicitly shell-quote both binary path and socket path when constructing the command string.

2. **Medium: No automated tests are planned for new command behavior**
- Current verification is mostly build/vet/manual.
- Risk: regressions in command wiring (`root -> server -> start`), session-exists handling, and output semantics are likely to slip in later refactors.
- Recommendation: add command-level tests for:
  - successful `server start` path (mock tmux boundary),
  - `mux.ErrSessionExists` path (exit 0 + message),
  - error path propagation.

3. **Medium: Existing session collision semantics are underspecified**
- Plan treats any `ErrSessionExists` as "already running".
- Risk: a pre-existing unrelated tmux session named `crabswarm` will be reported as running crabswarm server.
- Recommendation: document this as an accepted limitation, or add a lightweight validation step (e.g., check initial pane command/title marker) before reporting success-equivalent state.

## What Looks Good
- Reuse of existing `resolveSocketPath(cmd)` and `mux.ErrSessionExists` is appropriate.
- `server` parent command split mirrors existing command grouping structure.
- Manual verification scenarios are relevant and practical.

## Suggested Plan Amendments
- Add an explicit step for robust command construction (shell-escaped args).
- Add a test section with concrete test cases for `server start` behavior.
- Clarify/design collision behavior for existing `crabswarm` tmux sessions.
